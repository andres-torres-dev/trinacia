package campaign

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/facebook/auth"
	"bitbucket.org/backend/core/genetic"
	"bitbucket.org/backend/core/server"
	"bitbucket.org/backend/core/storage/campaigns"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

var (
	errFailRequest      = errors.New("failing request")
	errorFailStorage    = errors.New("failing storage")
	errorFailAuth       = errors.New("failing auth")
	errorFailingQuality = errors.New("failing quality function")
)

type createClient struct {
	// client failures variables are used to check code behavior when the client failes
	// to perform the specified request or invalid response is received
	failUnmarshal                         bool
	failCreateCampaignRequest             bool
	failGetAllCampaignsByAdAccountRequest bool
	failGetAdsetsRequest                  bool
	failGetNextPagingRequest              bool
	failGetAdsetTargetingRequest          bool
	failCreateAdsetRequest                bool
	failCreateAdCreativeRequest           bool
	failCreateAdRequest                   bool
	failGetAdsetInsightsRequest           bool

	// fail facebook operations with a facebook response error
	failCreateCampaign  bool
	failGetCampaigns    bool
	failGetAdsets       bool
	failAdsetsPaging    bool
	failAdsetsTargeting bool
	failAdset           bool
	failAdCreative      bool
	failAd              bool
	failAdsetInsights   bool

	// campaign checks
	status     string
	campaignID string
	adSetID    string
	adID       string
	creativeID string

	server.Client
	t *testing.T
}

type store struct {
	// storage failures variables are used to check code behavior when the storage failes
	// to perform the specified operation
	failGetSegment    bool
	failtSetSegment   bool
	failStoreCampaign bool

	segment []*genetic.Chromosome

	campaigns.Storage
	t *testing.T
}

type platformAuth struct {
	// fail authentication with an error
	failAuth bool
	// debug operation returns an invalid token
	invalidToken bool

	expected *entities.Facebook

	auth.Auth
	t *testing.T
}

type helper struct {
	// clientFailures array of failures
	// for the requests or unmarshal errors
	clientFailures []string
	// api errors
	campaignFailures []string
	// campaign config
	campaignID string

	// storageFailures is an array of
	// failures from the storage
	storageFailures []string
	// expectedSegment return from segment stored targeting
	expectedSegment []*genetic.Chromosome

	// authFailures is an array
	// of auth configurations
	authFailures []string
	// expectedAuth return from facebook auth
	expectedAuth *entities.Facebook

	quality func(c *genetic.Chromosome) (float64, error)

	t *testing.T
}

// configuartion helper to initialice Campaign interface
func (h *helper) testConfig(f *facebook) {
	h.t.Helper()

	defer func() {
		if r := recover(); r != nil {
			h.t.Fatalf("Unable to set client failure value: %v", r)
		}
	}()

	c := &createClient{
		t:          h.t,
		status:     f.status,
		campaignID: h.campaignID,
		adSetID:    "123412",
		creativeID: "12341234",
		adID:       "1234123",
	}
	cV := reflect.ValueOf(c)
	for _, clientFailure := range h.clientFailures {
		v := reflect.Indirect(cV).FieldByName(clientFailure)
		v.SetBool(true)
	}
	for _, apiFailure := range h.campaignFailures {
		v := reflect.Indirect(cV).FieldByName(apiFailure)
		v.SetBool(true)
	}
	f.client = c

	p := &store{
		t:       h.t,
		segment: h.expectedSegment,
	}
	pV := reflect.ValueOf(p)
	for _, storeFailure := range h.storageFailures {
		v := reflect.Indirect(pV).FieldByName(storeFailure)
		v.SetBool(true)
	}
	f.store = p

	a := &platformAuth{
		t:        h.t,
		expected: h.expectedAuth,
	}
	aV := reflect.ValueOf(a)
	for _, authFailure := range h.authFailures {
		v := reflect.Indirect(aV).FieldByName(authFailure)
		v.SetBool(true)
	}
	f.auth = a

	f.selection = genetic.New(func(c *genetic.Chromosome) (float64, error) {
		if c.Quality == 0.0 {
			return 0.0, errorFailingQuality
		}

		return c.Quality, nil
	})
}

func testUmarshal(t *testing.T, b []byte, out interface{}) {
	t.Helper()

	err := json.Unmarshal(b, out)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func (c *createClient) Post(u string, body io.Reader) (*http.Response, error) {
	requestURL, err := url.Parse(u)
	if err != nil {
		c.t.Fatal("Unable to parse request body from get request: ", err)
	}

	switch {
	case strings.Contains(requestURL.Path, "/campaigns"):
		if c.failCreateCampaignRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/adsets"):
		if c.failCreateAdsetRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/adcreatives"):
		if c.failCreateAdCreativeRequest {
			return nil, errFailRequest
		}
	case strings.Contains(requestURL.Path, "/ads"):
		if c.failCreateAdRequest {
			return nil, errFailRequest
		}
	}

	w := httptest.NewRecorder()
	handler := c.handler
	handler(w, requestURL, body)

	return w.Result(), nil
}

func (c *createClient) handler(w http.ResponseWriter, requestURL *url.URL, body io.Reader) {
	if c.failUnmarshal {
		io.WriteString(w, `{"status":200,}`)
		return
	}

	// read request bytes
	var b = []byte{}
	var buff = bytes.NewBuffer(b)
	_, err := buff.ReadFrom(body)
	if err != nil {
		c.t.Fatal("unable to read body", err)
	}
	req := buff.Bytes()

	var resp string
	switch {
	case strings.Contains(requestURL.Path, "/campaigns"):
		if c.failCreateCampaign {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
		newCampaign := &newCampaign{}
		testUmarshal(c.t, req, newCampaign)
		c.testCheckCampaignRequest(newCampaign)
		resp = fmt.Sprintf(`{"id":"%s"}`, c.campaignID)

	case strings.Contains(requestURL.Path, "/adsets"):
		if c.failAdset {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
		newAdSet := &newAdSet{}
		err := json.Unmarshal(req, newAdSet)
		if err != nil {
			c.t.Fatalf("err: %s", err)
		}
		c.testCheckAdSetRequest(newAdSet)
		resp = fmt.Sprintf(`{"id": "%s"}`, c.adSetID)

	case strings.Contains(requestURL.Path, "/adcreatives"):
		if c.failAdCreative {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
		newCreative := &newCreative{}
		testUmarshal(c.t, req, newCreative)
		c.testCheckCreativeRequest(newCreative)
		resp = fmt.Sprintf(`{"id":"%s"}`, c.creativeID)

	case strings.Contains(requestURL.Path, "/ads"):
		if c.failAd {
			io.WriteString(w, `{"error":{"message":"failing operation"}}`)
			return
		}
		newAd := &newAd{}
		testUmarshal(c.t, req, newAd)
		c.testCheckAdRequest(newAd)
		resp = fmt.Sprintf(`{"id":"%s"}`, c.adID)
	}

	io.WriteString(w, resp)
}

func (c *createClient) testCheckCampaignRequest(newCampaign *newCampaign) {
	switch {
	case newCampaign.Token == "":
		c.t.Fatal("Missing access token to create campaign")
	case newCampaign.Status != c.status:
		c.t.Fatal("Missmatch variable status to facebook status state")
	}
}

func (c *createClient) testCheckAdSetRequest(newAdSet *newAdSet) {
	switch {
	case newAdSet.Targeting == nil:
		c.t.Fatal("Fail to create targeting")
	case newAdSet.Name == "":
		c.t.Fatal("Missing adset name")
	case newAdSet.CampaignID == "":
		c.t.Fatal("Missing campaign id")
	case newAdSet.AccessToken == "":
		c.t.Fatal("Missing access token to create adset")
	case newAdSet.Status != c.status:
		c.t.Fatal("Missmatch variable status to global package")
	}
}

func (c *createClient) testCheckCreativeRequest(newCreative *newCreative) {
	switch {
	case newCreative.AccessToken == "":
		c.t.Fatal("Missing access token to create creative")
	case newCreative.Status != c.status:
		c.t.Fatal("Missmatch status variable")

	case newCreative.ObjectStroySpec.LinkData != linkData{}:
		if newCreative.ObjectStroySpec.LinkData.CallToAction.Type == "" {
			c.t.Fatal("Missing creative call to action type")
		}
		if newCreative.ObjectStroySpec.LinkData.Link == "" {
			c.t.Fatal("Missing ad creative link")
		}
		if newCreative.ObjectStroySpec.LinkData.ImageHash == "" {
			c.t.Fatal("Missing ad creative imash hash")
		}
		if newCreative.ObjectStroySpec.LinkData.CallToAction.Type == "LIKE_PAGE" {
			if newCreative.ObjectStroySpec.LinkData.Link != fmt.Sprintf("https://facebook.com/%s", newCreative.ObjectStroySpec.PageID) {
				c.t.Fatal("Invalid link for page like creative")
			}
			if newCreative.ObjectStroySpec.LinkData.CallToAction.Value.Page != newCreative.ObjectStroySpec.PageID {
				c.t.Fatal("Invalid object story spec value for a page like creative")
			}
		} else if newCreative.ObjectStroySpec.LinkData.CallToAction.Type != "NO_BUTTON" && newCreative.ObjectStroySpec.LinkData.CallToAction.Value.Link == "" {
			c.t.Fatal("Missing creative call to action link value")
		}

	case newCreative.ObjectStroySpec.VideoData != videoData{}:
		if newCreative.ObjectStroySpec.VideoData.VideoID == "" {
			c.t.Fatal("Missing ad creative video id")
		}
		if newCreative.ObjectStroySpec.VideoData.ImageHash == "" {
			c.t.Fatal("Missing creative video thumbnail image")
		}
		if newCreative.ObjectStroySpec.VideoData.CallToAction.Type == "" {
			c.t.Fatal("Missing ad creative call to action type")
		}
		if newCreative.ObjectStroySpec.VideoData.CallToAction.Type == "LIKE_PAGE" {
			if newCreative.ObjectStroySpec.LinkData.CallToAction.Value.Page != newCreative.ObjectStroySpec.PageID {
				c.t.Fatal("Invalid object story spec value for a page like creative")
			}
		} else if newCreative.ObjectStroySpec.VideoData.CallToAction.Type != "NO_BUTTON" && newCreative.ObjectStroySpec.VideoData.CallToAction.Value.Link == "" {
			c.t.Fatal("Missing creative call to action link value")
		}
	}
}

func (c *createClient) testCheckAdRequest(newAd *newAd) {
	switch {
	case newAd.Name == "":
		c.t.Fatal("Missing new ad name")
	case newAd.AdsetID == "":
		c.t.Fatal("Missing adset ID")
	case newAd.Status != c.status:
		c.t.Fatal("Missmatch status variable")
	case newAd.AccessToken == "":
		c.t.Fatal("Missing access token to create ad")
	case newAd.Creative.CreativeID == "":
		c.t.Fatal("Missing creative id to create ad")
	}
}

func (s *store) GetSegment(userID, segment string) ([]*genetic.Chromosome, error) {
	if s.failGetSegment {
		return nil, errorFailStorage
	}

	return s.segment, nil
}

func (s *store) SetSegment(userID, segment string, initialPopulation []*genetic.Chromosome) error {
	if s.failtSetSegment {
		return errorFailStorage
	}

	return nil
}

func (s *store) StoreCampaign(userID, platform, adAccount, segment string, c *entities.Campaign) error {
	if s.failStoreCampaign {
		return errorFailStorage
	}

	return nil
}

func (a *platformAuth) GetUser(userID string) (*entities.Facebook, bool, error) {
	if a.failAuth {
		return nil, false, errorFailAuth
	}
	if a.invalidToken {
		return a.expected, false, nil
	}

	return a.expected, true, nil
}
func TestCreate(t *testing.T) {
	var basicChromosome = []*genetic.Chromosome{
		{
			ID:      "1",
			Quality: 0.01,
			Root: &genetic.Gene{
				Children: []*genetic.Gene{
					{
						ID:    "Unique",
						Value: 1,
						Type:  "interests",
					},
				},
			},
		},
		{
			ID:      "2",
			Quality: 0.02,
			Root: &genetic.Gene{
				Children: []*genetic.Gene{
					{
						ID:    "UniqueTwo",
						Value: 1,
						Type:  "interests",
					},
				},
			},
		},
		{
			ID:      "3",
			Quality: 0.03,
			Root: &genetic.Gene{
				Children: []*genetic.Gene{
					{
						ID:    "UniqueThree",
						Value: 1,
						Type:  "interests",
					},
				},
			},
		},
		{
			ID:      "4",
			Quality: 0.4,
			Root: &genetic.Gene{
				Children: []*genetic.Gene{
					{
						ID:    "UniqueFour",
						Value: 1,
						Type:  "interests",
					},
				},
			},
		},
		{
			ID:      "5",
			Quality: 0.51,
			Root: &genetic.Gene{
				Children: []*genetic.Gene{
					{
						ID:    "UniqueFive",
						Value: 1,
						Type:  "interests",
					},
				},
			},
		},
	}
	cases := []struct {
		Name   string
		Helper *helper
		UserID string
		Req    *Request
		Error  error
	}{
		{
			Name: "Conversion Image Campaign",
			Helper: &helper{
				campaignID: "1234",
				expectedAuth: &entities.Facebook{
					ID:          "1234",
					AccessToken: "unicorn60",
				},
				expectedSegment: basicChromosome,
				t:               t,
			},
			UserID: "andres",
			Req: &Request{
				Name:              "testing C",
				Objective:         "CONVERSIONS",
				Budget:            "3000",
				SpecialAdCategory: []string{},
				Segment:           "techUnicorn",
				MutationRate:      0.01,
				StartTime:         time.Now().String(),
				EndTime:           time.Now().Add(time.Hour * 365).String(),
				Location: geolocation{
					Countries: []string{"CO"},
				},
				Gender: [2]int{1, 1},
				AgeMax: 45,
				AgeMin: 25,
				Page: entities.Page{
					Category:    "Marketing",
					Name:        "Trinacia",
					ID:          "trinacia official",
					AccessToken: "1234",
				},
				CreativeName: "first campaign",
				MediaURL:     "https://trinacia.com",
				ImageHash:    "123412341234123",
				Message:      "start now!",
				CallToAction: callToAction{
					Type: "BUY_NOW",
					Value: callToActionValue{
						Link: "https://trinacia.com",
						Page: "trinacia official",
					},
				},
				AdAccount: "act_1234123",
			},
			Error: nil,
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
		campaign := New(sess, tc.Helper.testConfig)
		_, err := campaign.Create(tc.UserID, tc.Req)
		assert.Equal(tc.Error, err)
	}
}
