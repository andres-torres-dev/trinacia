package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/facebook/internal"
	"bitbucket.org/backend/core/logger"
	"bitbucket.org/backend/core/server"
	platform "bitbucket.org/backend/core/storage/facebook"
	"bitbucket.org/backend/core/storage/user"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Auth methods for facebook
type Auth interface {
	AuthUser(code, userID string) (*entities.Facebook, error)
	GetUser(userID string) (*entities.Facebook, bool, error)
}

type facebook struct {
	userStore     user.Storage
	platformStore platform.Storage
	client        server.Client
}

// New auth facebook interface
func New(sess *session.Session, config ...func(*facebook)) Auth {
	f := &facebook{
		platformStore: platform.NewFacebook(sess),
		client:        server.New(),
	}

	for _, fn := range config {
		fn(f)
	}

	return f
}

var (
	// ErrorNilCode the code to exchange for a facebook access token is nil
	ErrorNilCode = errors.New("The code to exchange for a facebook access token is nil")
	// ErrorNilUser the request doesn't provide the user id
	ErrorNilUser = errors.New("Nil user id")
)

type debugAccessToken struct {
	AppID       string `json:"app_id"`
	Type        string `json:"type"`
	Application string `json:"application"`
	Valid       bool   `json:"is_valid"`
	ExpiresAt   int    `json:"expires_at"`
	IssuedAt    int    `json:"issued_at"`
	Metadata    struct {
		Sso string `json:"sso"`
	} `json:"metadata"`
	Scopes []string `json:"scopes"`
	UserID string   `json:"user_id"`
}

func (f *facebook) AuthUser(code, userID string) (*entities.Facebook, error) {
	if userID == "" {
		return nil, &logger.Error{
			Level: "Warning",
			Err:   ErrorNilUser,
		}
	}
	if code == "" {
		return nil, &logger.Error{
			Level: "Warning",
			Err:   ErrorNilCode,
		}
	}

	var (
		e *entities.Facebook
	)

	token, err := f.exchangeCode(code)
	if err != nil {
		return nil, err
	}

	debug, err := f.debugToken(token)
	if err != nil {
		return nil, err
	}

	pages, err := f.getPages(token, debug.UserID)
	if err != nil {
		return nil, err
	}

	adAccounts, err := f.getAdAccounts(token, debug.UserID)
	if err != nil {
		return nil, err
	}

	e = &entities.Facebook{
		ID:          debug.UserID,
		Pages:       pages,
		AdAccounts:  adAccounts,
		AccessToken: token,
	}

	err = f.platformStore.StoreFacebook(userID, e)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to store facebook information for user",
		}
	}

	return e, nil
}

func (f *facebook) exchangeCode(code string) (string, error) {
	var (
		result = struct {
			Token string                  `json:"access_token"`
			Error *internal.FacebookError `json:"error"`
		}{}
	)

	uV := url.Values{}
	uV.Add("client_id", os.Getenv("clientID"))
	uV.Add("redirect_uri", os.Getenv("redirectURL"))
	uV.Add("client_secret", os.Getenv("clientSecret"))
	uV.Add("code", code)
	u := internal.SetURL("oauth/access_token", uV)

	resp, err := f.client.Get(u)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to perform get request to exchange authentication code",
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to read response body during code exchange for facebook authentication",
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to unmarshal result during code exchange for facebook authentication",
		}
	}

	if result.Error != nil {
		return "", &logger.Error{
			Level:   "Warning",
			Err:     result.Error,
			Message: "Response to the code exchange for a facebook  authentication contained an error",
		}
	}

	return result.Token, nil
}

func (f *facebook) getPages(t, id string) ([]entities.Page, error) {
	var (
		result = struct {
			Pages []entities.Page         `json:"data"`
			Error *internal.FacebookError `json:"error"`
		}{}
	)

	uV := url.Values{}
	uV.Add("access_token", t)
	uV.Add("fields", "id,name,category,access_token")
	u := internal.SetURL(fmt.Sprintf("%s/accounts", id), uV)

	resp, err := f.client.Get(u)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to perform get request to retrieve user facebook pages",
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to read response body during the retrieval of facebook pages",
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to unmarshal result during the retrieval of facebook pages",
		}
	}

	if result.Error != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Err:     result.Error,
			Message: "Response to retrieve facebook pages contained an error",
		}
	}

	err = f.getInstagram(result.Pages)
	if err != nil {
		return nil, err
	}

	return result.Pages, nil
}

func (f *facebook) getInstagram(pages []entities.Page) error {
	type result struct {
		Instagram []entities.Instagram    `json:"data"`
		Error     *internal.FacebookError `json:"error"`
	}

	for i, p := range pages {
		re := result{}
		uV := url.Values{}
		uV.Add("access_token", p.AccessToken)
		uV.Add("fields", "id,username")
		u := internal.SetURL(fmt.Sprintf("%s/instagram_accounts", p.ID), uV)

		resp, err := f.client.Get(u)
		if err != nil {
			return &logger.Error{
				Level:   "Panic",
				Err:     err,
				Message: "Unable to make get request during the retrieval of a page instagram",
			}
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return &logger.Error{
				Level:   "Panic",
				Err:     err,
				Message: "Unable to read response body during the retrieval of a page instagram",
			}
		}
		err = json.Unmarshal(b, &re)
		if err != nil {
			return &logger.Error{
				Level:   "Panic",
				Err:     err,
				Message: "Unable to unmarshal result during the retrieval of a page instagram",
			}
		}
		if re.Error != nil {
			return &logger.Error{
				Level:   "Error",
				Err:     re.Error,
				Message: "Response to retrieve a page instagram contained an error",
			}
		}
		pages[i].Instagram = re.Instagram
	}

	return nil
}

func (f *facebook) getAdAccounts(t, id string) ([]entities.AdAccount, error) {
	var (
		result = struct {
			AdAccounts []entities.AdAccount    `json:"data"`
			Error      *internal.FacebookError `json:"error"`
		}{}
	)

	uV := url.Values{}
	uV.Add("access_token", t)
	uV.Add("fields", "id,account_id,name,currency")
	u := internal.SetURL(fmt.Sprintf("%s/adaccounts", id), uV)

	resp, err := f.client.Get(u)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to perform get request to retrieve user ad accounts",
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to read response body during the retrieval of user ad accounts",
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to unmarshal result during the retrieval of user ad accounts",
		}
	}
	if result.Error != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Err:     result.Error,
			Message: "Response to retrieve a user's ad accounts contained an error",
		}
	}

	return result.AdAccounts, nil
}

func (f *facebook) debugToken(t string) (*debugAccessToken, error) {
	var (
		result = struct {
			DebugAccessToken *debugAccessToken       `json:"data"`
			Error            *internal.FacebookError `json:"error"`
		}{}
	)

	uV := url.Values{}
	uV.Add("input_token", t)
	uV.Add("access_token", os.Getenv("appToken"))
	u := internal.SetURL("debug_token", uV)

	resp, err := f.client.Get(u)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to make get request during the debugging of a facebook access token",
		}
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to read response body during the debugging of a facebook access token",
		}
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Panic",
			Err:     err,
			Message: "Unable to unmarshal result during the debugging of a facebook access token",
		}
	}

	if result.Error != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Err:     result.Error,
			Message: "Response to debug a user's token contained an error",
		}
	}

	return result.DebugAccessToken, nil
}

func (f *facebook) GetUser(userID string) (*entities.Facebook, bool, error) {
	if userID == "" {
		return nil, false, &logger.Error{
			Level: "Warning",
			Err:   ErrorNilUser,
		}
	}
	e, err := f.platformStore.GetFacebook(userID)
	if err != nil {
		return nil, false, &logger.Error{
			Level:   "Panic",
			Message: "Unable to Get Facebook Data for user",
			Err:     err,
		}
	}
	if e.AccessToken == "" {
		return nil, false, nil
	}
	debug, err := f.debugToken(e.AccessToken)
	if err != nil {
		return nil, false, err
	}

	return e, debug.Valid, nil
}
