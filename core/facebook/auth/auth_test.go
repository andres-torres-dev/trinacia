package auth

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/facebook/internal"
	"bitbucket.org/backend/core/logger"
	"bitbucket.org/backend/core/server"
	platform "bitbucket.org/backend/core/storage/facebook"
	"github.com/stretchr/testify/assert"
)

var (
	// error to provide after failing client request
	errFailRequest = errors.New("request failed by client")
	// error to provide after failing storage
	errFailStorage = errors.New("request failed by store")
)

type client struct {
	// client request errors
	FailExchangeCodeRequest  bool
	FailDebugTokenRequest    bool
	FailGetPagesRequest      bool
	FailGetInstagramRequest  bool
	FailGetAdAccountsRequest bool

	// facebook api error
	FailExchangeCode  bool
	FailDebugToken    bool
	FailGetPages      bool
	FailGetInstagram  bool
	FailGetAdAccounts bool

	// fail unmarshal operation
	// by sending malformed json response
	FailUnmarshal bool

	server.Client
	t *testing.T
}

type platformStore struct {
	FailStoreFacebook bool
	FailGetFacebook   bool

	// expected retrieved value from get operation
	expected *entities.Facebook

	platform.Storage
	t *testing.T
}

func (s *platformStore) StoreFacebook(userID string, f *entities.Facebook) error {
	if s.FailStoreFacebook {
		return errFailStorage
	}

	return nil
}

func (s *platformStore) GetFacebook(userID string) (*entities.Facebook, error) {
	if s.FailGetFacebook {
		return nil, errFailStorage
	}

	return s.expected, nil
}

func (c *client) Get(u string) (*http.Response, error) {
	requestURL, err := url.Parse(u)
	if err != nil {
		c.t.Fatal("Unable to parse request body from get request: ", err)
	}

	switch {
	case requestURL.Path == "/v8.0/oauth/access_token":
		if c.FailExchangeCodeRequest {
			return nil, errFailRequest
		}
	case requestURL.Path == "/v8.0/debug_token":
		if c.FailDebugTokenRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/accounts"):
		if c.FailGetPagesRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/instagram_accounts"):
		if c.FailGetInstagramRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/adaccounts"):
		if c.FailGetAdAccountsRequest {
			return nil, errFailRequest
		}
	}

	w := httptest.NewRecorder()
	handler := c.authHandler
	handler(w, requestURL)

	return w.Result(), nil
}

func (c *client) authHandler(w http.ResponseWriter, requestURL *url.URL) {
	if c.FailUnmarshal {
		io.WriteString(w, `{"status":200,}`)
		return
	}

	// handle controlled failures
	switch {
	case requestURL.Path == "/v8.0/oauth/access_token":
		if c.FailExchangeCode {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}

	case requestURL.Path == "/v8.0/debug_token":
		if c.FailDebugToken {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
	case strings.Contains(requestURL.Path, "/accounts"):
		if c.FailGetPages {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}

	case strings.Contains(requestURL.Path, "/instagram_accounts"):
		if c.FailGetInstagram {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}

	case strings.Contains(requestURL.Path, "/adaccounts"):
		if c.FailGetAdAccounts {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
	}
	// read data from files
	if _, err := os.Open(fmt.Sprintf("test-fixtures%s.json", requestURL.Path)); err != nil {
		io.WriteString(w, fmt.Sprintf(`{
			"error": {
			  "message": "(#803) Some of the aliases you requested do not exist: %s",
			  "type": "OAuthException",
			  "code": 803,
			  "fbtrace_id": "AOFibG3hmX4AtsTdmEelqLn"
			}
		}`, requestURL.Path))
		return
	}

	b, err := ioutil.ReadFile(fmt.Sprintf("test-fixtures%s.json", requestURL.Path))
	if err != nil {
		c.t.Fatalf("err: %s", err)
	}

	io.WriteString(w, string(b))
}

type helper struct {
	// clientFailures array of failures
	// for the requests or unmarshal errors
	clientFailures []string

	// facebookFailures is an array
	// of errors from facebook api operations
	facebookFailures []string

	// storageFailures is an array of
	// failures from the storage
	storageFailures []string
	// expected return value from storage
	// get facebook operation
	expected *entities.Facebook

	t *testing.T
}

// configuration helper to initialice Auth interface
func (h *helper) testConfig(f *facebook) {
	h.t.Helper()

	defer func() {
		if r := recover(); r != nil {
			h.t.Fatalf("Unable to set client failure value: %v", r)
		}
	}()

	c := &client{
		t: h.t,
	}
	cV := reflect.ValueOf(c)
	for _, clientFailure := range h.clientFailures {
		v := reflect.Indirect(cV).FieldByName(clientFailure)
		v.SetBool(true)
	}
	for _, apiFailure := range h.facebookFailures {
		v := reflect.Indirect(cV).FieldByName(apiFailure)
		v.SetBool(true)
	}
	f.client = c

	p := &platformStore{
		t:        h.t,
		expected: h.expected,
	}
	pV := reflect.ValueOf(p)
	for _, storeFailure := range h.storageFailures {
		v := reflect.Indirect(pV).FieldByName(storeFailure)
		v.SetBool(true)
	}
	f.platformStore = p
}

func TestAuthUser(t *testing.T) {
	cases := []struct {
		Name, Code, UserID string
		clientFailures     []string
		facebookFailures   []string
		storageFailures    []string
		Expected           *entities.Facebook
		Error              error
	}{
		{
			Name:   "Auth User",
			Code:   "AQB2JdlAuasdfasdfasdfqwefwqeejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			Expected: &entities.Facebook{
				ID: "2730207623713666",
				Pages: []entities.Page{
					{
						Category:    "Accessories",
						AccessToken: "EAAHu3c2xquQBAJ3fU0ZAjo4E3BEAn5AaBZA37KlswrUMMAZAuTkWGmwMHfkljacanwUhRQT8bL7o0FbVUBltTjraXic9rTzOZB7ZAIqa6MVshlOjOstaIfs315MZBnfvk3ub24VtzHD4ITpfMSWS2woo4q0ZBDu9zPObq96WQWBGPO3NVaaApgPNR9SybcR9MyPnFxcwieVK27OrxskVax8",
						ID:          "101564201278325",
						Name:        "The gossip corner",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
					{
						Category:    "Food Delivery Service",
						AccessToken: "EAAHu3c22fweqfqwefasdfasdApqAjsLgIeROCSgfxut1cXEtCYbRCyKVeKNdPKZBzkZCDwlUT0j50oWTFRweE48SmN5RZCp4DUZAFxcQlZCUNJSzDCtji8ldfqrfiMjWkwBzzfExddhhoeEETQ5OJniMMA8mkC85zJR4isYkmVidGL7Cwm0mFfISFEWuuRNZBbwfZANnkYaV7lqZAR4TYw",
						ID:          "694235647641117",
						Name:        "Ignis Cuisine",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "656522844415498",
						ID:        "act_656522844415498",
					},
				},
				AccessToken: "EasdfasdfasdfasdfasdjUfPSZB5VAnpTfplKdYKYRMdf9L3kRzemFgquPHZBZB7ZCEgSUlLqqvobDpslavSX6hWsyjo3xrHuc40OYvRdR2ZCcmDUiyZAVsHgRgbBLiSCH5M2bNpv6nR7rAqZBurkvTNo4JGdSxqyKQNNX9yfoPaWKHiRY6oZAwZBJGmrRFLZCkSMBNyVpCZAWYMdZBf",
			},
			Error: nil,
		},
		{
			Name:     "Missing User",
			Code:     "AQB2JdlAuasdfasdfwqefasdfsajaZRRihLKxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID:   "",
			Expected: nil,
			Error: &logger.Error{
				Level: "Warning",
				Err:   ErrorNilUser,
			},
		},
		{
			Name:     "Missing Code",
			Code:     "",
			UserID:   "12345",
			Expected: nil,
			Error: &logger.Error{
				Level: "Warning",
				Err:   ErrorNilCode,
			},
		},
		{
			Name:   "Fail Exchange Code Request",
			Code:   "AQB2asdfasdfwqerf234efdsaLKxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailExchangeCodeRequest",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailRequest,
				Message: "Unable to perform get request to exchange authentication code",
			},
		},
		{
			Name:   "Fail API Operation to Exchange Code",
			Code:   "AQB2JdlAuDV61cIy4fsadfasdasdfasdfasdfas23ejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailExchangeCode",
			},
			Expected: nil,
			Error: &logger.Error{
				Level: "Warning",
				Err: &internal.FacebookError{
					Message: "failing operation",
				},
				Message: "Response to the code exchange for a facebook  authentication contained an error",
			},
		},
		{
			Name:   "Fail Debug Token Request",
			Code:   "AQB2JdlAuDVasdfasdfasdf7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailDebugTokenRequest",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Message: "Unable to make get request during the debugging of a facebook access token",
				Err:     errFailRequest,
			},
		},
		{
			Name:   "Fail API Operation to Debug Token",
			Code:   "AQB2JdlAuDV61cIyvJKH3eTlSixcGsajaZRRihLKasdfasdfasdfVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailDebugToken",
			},
			Expected: nil,
			Error: &logger.Error{
				Level: "Error",
				Err: &internal.FacebookError{
					Message: "failing operation",
				},
				Message: "Response to debug a user's token contained an error",
			},
		},
		{
			Name:   "Fail Get Pages Request",
			Code:   "AQB2JdlAuDV61cIyvJKH3eTlSasdfasdfasdfasdr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetPagesRequest",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailRequest,
				Message: "Unable to perform get request to retrieve user facebook pages",
			},
		},
		{
			Name:   "Fail API Operation to Get Pages",
			Code:   "AQB2JdlAuDVasdfasdfasdB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetPages",
			},
			Expected: nil,
			Error: &logger.Error{
				Level: "Error",
				Err: &internal.FacebookError{
					Message: "failing operation",
				},
				Message: "Response to retrieve facebook pages contained an error",
			},
		},
		{
			Name:   "Fail Get Instagram Request",
			Code:   "AQB2JdlAuDVasdfasdfasdfajaZRRihLKxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetInstagramRequest",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailRequest,
				Message: "Unable to make get request during the retrieval of a page instagram",
			},
		},
		{
			Name:   "Fail API Operation to Get Instagram",
			Code:   "AQB2JdlAuDV61cIyvJKH3easdfasdfejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetInstagram",
			},
			Expected: nil,
			Error: &logger.Error{
				Level: "Error",
				Err: &internal.FacebookError{
					Message: "failing operation",
				},
				Message: "Response to retrieve a page instagram contained an error",
			},
		},
		{
			Name:   "Fail Get Ad Accounts Request",
			Code:   "AQB2asdfasdfasdfasdfGsajaZRRihLKxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetAdAccountsRequest",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailRequest,
				Message: "Unable to perform get request to retrieve user ad accounts",
			},
		},
		{
			Name:   "Fail API Operation to Get Ad Accounts",
			Code:   "AQB2JdlAuDasdfasdfajaZRRihLKxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			clientFailures: []string{
				"FailGetAdAccounts",
			},
			Expected: nil,
			Error: &logger.Error{
				Level: "Error",
				Err: &internal.FacebookError{
					Message: "failing operation",
				},
				Message: "Response to retrieve a user's ad accounts contained an error",
			},
		},
		{
			Name:   "Fail Store Facebook",
			Code:   "AQB2JdlAuDV61cIyvJKH3eTlSasdfasdfasdfxmejB7p7nwtn9nInwoR2kYrr4GmCVhoksVwsAhs4rrkYYdpize8PrDMmjAQBQUDKjLLqWnh4CdJ_3F4MW9ezV9z79oOBKCHjNayZr_FbQfSrBNZKkkGdHt1",
			UserID: "123451432",
			storageFailures: []string{
				"FailStoreFacebook",
			},
			Expected: nil,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailStorage,
				Message: "Unable to store facebook information for user",
			},
		},
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			h := &helper{
				t:                t,
				clientFailures:   tc.clientFailures,
				facebookFailures: tc.facebookFailures,
				storageFailures:  tc.storageFailures,
			}
			auth := New(sess, h.testConfig)
			f, err := auth.AuthUser(tc.Code, tc.UserID)
			assert.Equal(tc.Expected, f)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetUser(t *testing.T) {
	cases := []struct {
		Name   string
		UserID string
		// defaultFacebook is used for the storage return
		defaultFacebook  *entities.Facebook
		ExpectedFacebook *entities.Facebook
		ExpectedValidity bool
		Error            error
		clientFailures   []string
		storageFailures  []string
	}{
		{
			Name:   "Get User Valid Token",
			UserID: "12341234",
			ExpectedFacebook: &entities.Facebook{
				ID: "2730207623713666",
				Pages: []entities.Page{
					{
						Category:    "Accessories",
						AccessToken: "asdfasdfsadlswrUMMAZAuTkWGmwMHfkljacanwUhRQT8bL7o0FbVUBltTjraXic9rTzOZB7ZAIqa6MVshlOjOstaIfs315MZBnfvk3ub24VtzHD4ITpfMSWS2woo4q0ZBDu9zPObq96WQWBGPO3NVaaApgPNR9SybcR9MyPnFxcwieVK27OrxskVax8",
						ID:          "101564201278325",
						Name:        "The gossip corner",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
					{
						Category:    "Food Delivery Service",
						AccessToken: "EAAHu3c2xquQBAC2ZARBNcwSw4xZBApqAjsLgIeROCSgfxut1cXEtCYbRCyKVeKNdPKZBzkZCDwlUT0j50oWTFRweE48SmN5RZCp4DUZAFxcQlZCUNJSzDCtji8ldfqrfiMjWkwBzzfExddhhoeEETQ5OJniMMA8mkC85zJR4isYkmVidGL7Cwm0mFfISFEWuuRNZBbwfZANnkYaV7lqZAR4TYw",
						ID:          "694235647641117",
						Name:        "Ignis Cuisine",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "656522844415498",
						ID:        "act_656522844415498",
					},
				},
				AccessToken: "EAAHuasdfasdNZBMTd8nzLxSv4jhjUfPSZB5VAnpTfplKdYKYRMdf9L3kRzemFgquPHZBZB7ZCEgSUlLqqvobDpslavSX6hWsyjo3xrHuc40OYvRdR2ZCcmDUiyZAVsHgRgbBLiSCH5M2bNpv6nR7rAqZBurkvTNo4JGdSxqyKQNNX9yfoPaWKHiRY6oZAwZBJGmrRFLZCkSMBNyVpCZAWYMdZBf",
			},
			ExpectedValidity: true,
			Error:            nil,
		},
		{
			Name:             "Missing User ID",
			UserID:           "",
			ExpectedFacebook: nil,
			ExpectedValidity: false,
			Error: &logger.Error{
				Level: "Warning",
				Err:   ErrorNilUser,
			},
		},
		{
			Name:             "Fail to Get Data from Data Base",
			UserID:           "1234123412",
			ExpectedFacebook: nil,
			ExpectedValidity: false,
			Error: &logger.Error{
				Level:   "Panic",
				Message: "Unable to Get Facebook Data for user",
				Err:     errFailStorage,
			},
			storageFailures: []string{
				"FailGetFacebook",
			},
		},
		{
			Name:             "Facebook not found in Data Base",
			UserID:           "14312341234",
			defaultFacebook:  &entities.Facebook{},
			ExpectedFacebook: nil,
			ExpectedValidity: false,
			Error:            nil,
		},
		{
			Name:   "Fail to Debug Token",
			UserID: "12341234",
			defaultFacebook: &entities.Facebook{
				ID: "2730207623713666",
				Pages: []entities.Page{
					{
						Category:    "Accessories",
						AccessToken: "EAAHu3c2xquQBAJ3fU0ZAjo4E3BEAn5AaBZA37KlswrUMMAZAuTkWGmwMHfkljacanwUhRQT8bL7o0FbVUBltTjraXic9rTzOZB7ZAIqa6MVshlOjOstaIfs315MZBnfvk3ub24VtzHD4ITpfMSWS2woo4q0ZBDu9zPObq96WQWBGPO3NVaaApgPNR9SybcR9MyPnFxcwieVK27OrxskVasdf",
						ID:          "101564201278325",
						Name:        "The gossip corner",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
					{
						Category:    "Food Delivery Service",
						AccessToken: "EAAHu3c2xquQBAC2ZARBNcwSw4xZBApqAjsLgIeROCSgfxut1cXEtCYbRCyKVeKNdPKZBzkZCDwlUT0j50oWTFRweE48SmN5RZCp4DUZAFxcQlZCUNJSzDCtji8ldfqrfiMjWkwBzzfExddhhoeEE2wer23fasdMA8mkC85zJR4isYkmVidGL7Cwm0mFfasdfasdfasdfZANnkYaV7lqZAR4TYw",
						ID:          "694235647641117",
						Name:        "Ignis Cuisine",
						Instagram: []entities.Instagram{
							{
								ID:   "3240241866073010",
								Name: "thegossipocorner",
							},
						},
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "656522844415498",
						ID:        "act_656522844415498",
					},
				},
				AccessToken: "EAAHasdfasdfuQBANZBMTd8nzLxSv4jhjUfPSZB5VAnpTfplKdYKYRMdf9L3kRzemFgquPHZBZB7ZCEgSUlLqqvobDpslavSX6hWsyjo3xrHuc40OYvRdR2ZCcmDUiyZAVsHgRgbBLiSCH5M2bNpv6nR7rAqZBurkvTNo4JGdSxqyKQNNX9yfoPaWKHiRY6oZAwZBJGmrRFLZCkSMBNyVpCZAWYMdZBf",
			},
			ExpectedFacebook: nil,
			ExpectedValidity: false,
			Error: &logger.Error{
				Level:   "Panic",
				Err:     errFailRequest,
				Message: "Unable to make get request during the debugging of a facebook access token",
			},
			clientFailures: []string{
				"FailDebugTokenRequest",
			},
		},
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			h := &helper{
				t:               t,
				expected:        tc.ExpectedFacebook,
				clientFailures:  tc.clientFailures,
				storageFailures: tc.storageFailures,
			}
			if tc.defaultFacebook != nil {
				h.expected = tc.defaultFacebook
			}

			auth := New(sess, h.testConfig)
			f, valid, err := auth.GetUser(tc.UserID)
			assert.Equal(tc.ExpectedFacebook, f)
			assert.Equal(tc.ExpectedValidity, valid)
			assert.Equal(tc.Error, err)
		})
	}
}
