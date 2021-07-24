package campaigns

import (
	"testing"
	"time"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/genetic"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func testCreateCampaign(t *testing.T, userID, platform, adAccount, segment string, c *entities.Campaign) {
	t.Helper()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := New(sess)
	err = storage.StoreCampaign(userID, platform, adAccount, segment, c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testDeleteItem(t *testing.T, partition, key string) {
	t.Helper()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	svc := dynamodb.New(sess)
	in := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(partition),
			},
			"key": {
				S: aws.String(key),
			},
		},
	}
	if _, err := svc.DeleteItem(in); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestStoreCampaign(t *testing.T) {
	cases := []struct {
		Name, UserID, Platform, AdAccount, Segment string
		Campaign                                   *entities.Campaign
		Error                                      error
	}{
		{
			Name:      "Correct",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: nil,
		},
		{
			Name:      "Missing User ID",
			UserID:    "",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorMissingUserID,
		},
		{
			Name:      "Missing Platform",
			UserID:    "1234",
			Platform:  "",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorMissingPlatform,
		},
		{
			Name:      "Missing Ad Account",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorMissingAdAccount,
		},
		{
			Name:      "Correct",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorMissingSegment,
		},
		{
			Name:      "Nil Pointer Campaign reference",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign:  nil,
			Error:     ErrorInvalidCampaign,
		},
		{
			Name:      "Missing Campaign ID",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorInvalidCampaign,
		},
		{
			Name:      "Missing Start Time",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: "",
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorInvalidCampaign,
		},
		{
			Name:      "Missing End Time",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   "",
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorInvalidCampaign,
		},
		{
			Name:      "Missing Budget",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorInvalidCampaign,
		},
		{
			Name:      "Missing Targeting",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: nil,
				Media: []entities.Media{
					{},
				},
			},
			Error: ErrorInvalidCampaign,
		},
		{
			Name:      "Missing Media",
			UserID:    "1234",
			Platform:  "facebook",
			AdAccount: "ac_1234",
			Segment:   "Unicorn",
			Campaign: &entities.Campaign{
				ID:        "campaign1234",
				Budget:    "1 bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{},
				},
				Media: nil,
			},
			Error: ErrorInvalidCampaign,
		},
	}
	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := New(sess)

	for _, tc := range cases {
		if tc.Error == nil {
			defer testDeleteItem(t, "campaigns", tc.Campaign.ID)
		}
		t.Run(tc.Name, func(t *testing.T) {
			err := storage.StoreCampaign(tc.UserID, tc.Platform, tc.AdAccount, tc.Segment, tc.Campaign)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetCampaign(t *testing.T) {
	cases := []struct {
		Name     string
		ID       string
		Expected *entities.Campaign
		Error    error
	}{
		{
			Name: "Correct",
			ID:   "1234",
			Expected: &entities.Campaign{
				ID:        "1234",
				Budget:    "1bn",
				StartTime: time.Now().String(),
				EndTime:   time.Now().Add(time.Hour * 365).String(),
				Targeting: []*genetic.Chromosome{
					{
						ID: "test",
					},
				},
				Media: []entities.Media{
					{},
				},
			},
		},
		{
			Name:     "Missing ID",
			ID:       "",
			Expected: nil,
			Error:    ErrorMissingCampaignID,
		},
		{
			Name:     "Unable To Find Campaign",
			ID:       "12345",
			Expected: nil,
			Error:    ErrorUnableToFindCampaign,
		},
	}
	// create test campaigns
	for _, tc := range cases {
		if tc.Error == nil {
			testCreateCampaign(t, "testUser", "testPlatform", "testAdAccount", "testSegment", tc.Expected)
			defer testDeleteItem(t, "campaigns", tc.Expected.ID)
		}
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := New(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			c, err := storage.GetCampaign(tc.ID)
			assert.Equal(tc.Expected, c)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetUserCampaigns(t *testing.T) {
	type campaignData struct {
		AdAccount, Segment string
		Campaign           *entities.Campaign
	}
	cases := []struct {
		Name     string
		UserID   string
		Platform string
		Campaign []campaignData
		Expected map[string][]string
		Error    error
	}{
		{
			Name:     "Zero Campaigns",
			UserID:   "123",
			Platform: "testPlatform",
			Campaign: nil,
			Expected: map[string][]string{},
			Error:    nil,
		},
		{
			Name:     "Single Campaign",
			UserID:   "12345",
			Platform: "testPlatform",
			Campaign: []campaignData{
				{
					AdAccount: "testAdAccount",
					Segment:   "testSegment",
					Campaign: &entities.Campaign{
						ID:        "1234",
						Budget:    "1bn",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{
				"testPlatform": []string{"1234"},
			},
			Error: nil,
		},
		{
			Name:     "Multiple Campaigns",
			UserID:   "123456",
			Platform: "testPlatform",
			Campaign: []campaignData{
				{
					AdAccount: "testAdAccount",
					Segment:   "testSegment",
					Campaign: &entities.Campaign{
						ID:        "12345",
						Budget:    "1bn",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
				{
					AdAccount: "testAdAccount",
					Segment:   "testSegment",
					Campaign: &entities.Campaign{
						ID:        "123456",
						Budget:    "1bn",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{
				"testPlatform": []string{"12345", "123456"},
			},
			Error: nil,
		},
		{
			Name:     "Missing User ID",
			UserID:   "",
			Platform: "testPlatform",
			Campaign: nil,
			Expected: nil,
			Error:    ErrorMissingUserID,
		},
	}
	for _, tc := range cases {
		if tc.Error == nil {
			for _, campaign := range tc.Campaign {
				testCreateCampaign(t, tc.UserID, tc.Platform, campaign.AdAccount, campaign.Segment, campaign.Campaign)
				defer testDeleteItem(t, "campaigns", campaign.Campaign.ID)
			}

		}
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := New(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			c, err := storage.GetUserCampaigns(tc.UserID)
			assert.Equal(tc.Expected, c)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetActiveCampaigns(t *testing.T) {
	type campaignData struct {
		UserID, AdAccount, Segment string
		Campaign                   *entities.Campaign
	}
	cases := []struct {
		Name      string
		Platform  string
		Campaigns []campaignData
		Expected  map[string][]string
		Error     error
	}{
		{
			Name:     "Zero Active Campaigns",
			Platform: "test",
			Campaigns: []campaignData{
				{
					UserID:    "123",
					AdAccount: "123",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "123",
						Budget:    "1234",
						StartTime: time.Now().Add(-time.Hour * 365).String(),
						EndTime:   time.Now().Add(-time.Hour * 48).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{},
			Error:    nil,
		},
		{
			Name:     "One Active Campaign One User",
			Platform: "test1",
			Campaigns: []campaignData{
				{
					UserID:    "123",
					AdAccount: "123",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "1234",
						Budget:    "1234",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{
				"123": []string{"1234"},
			},
			Error: nil,
		},
		{
			Name:     "Multiple Active Campaigns One User",
			Platform: "test2",
			Campaigns: []campaignData{
				{
					UserID:    "1234",
					AdAccount: "1234",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "12345",
						Budget:    "1234",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
				{
					UserID:    "1234",
					AdAccount: "1234",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "123456",
						Budget:    "1234",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{
				"1234": []string{"12345", "123456"},
			},
			Error: nil,
		},
		{
			Name:     "Multiple Campaigns Multiple Users",
			Platform: "test3",
			Campaigns: []campaignData{
				{
					UserID:    "1234",
					AdAccount: "1234",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "1234567",
						Budget:    "1234",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
				{
					UserID:    "12345",
					AdAccount: "1234",
					Segment:   "test1",
					Campaign: &entities.Campaign{
						ID:        "12345678",
						Budget:    "1234",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: map[string][]string{
				"1234":  []string{"1234567"},
				"12345": []string{"12345678"},
			},
			Error: nil,
		},
		{
			Name:      "Missing Platform",
			Platform:  "",
			Campaigns: nil,
			Expected:  nil,
			Error:     ErrorMissingPlatform,
		},
	}
	for _, tc := range cases {
		if tc.Error == nil {
			for _, campaign := range tc.Campaigns {
				testCreateCampaign(t, campaign.UserID, tc.Platform, campaign.AdAccount, campaign.Segment, campaign.Campaign)
				defer testDeleteItem(t, "campaigns", campaign.Campaign.ID)
			}
		}
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	storage := New(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			c, err := storage.GetActiveCampaigns(tc.Platform)
			assert.Equal(tc.Expected, c)
			if tc.Error != err {
				t.Fatal(err)
			}
		})
	}
}

func TestGetSegmentCampaigns(t *testing.T) {
	type campaignData struct {
		Platform, AdAccount string
		Campaign            *entities.Campaign
	}
	cases := []struct {
		Name      string
		UserID    string
		Segment   string
		Campaigns []campaignData
		Expected  []string
		Error     error
	}{
		{
			Name:      "Zero Campaigns Under Segment",
			UserID:    "1234",
			Segment:   "Test",
			Campaigns: nil,
			Expected:  []string{},
			Error:     nil,
		},
		{
			Name:    "Single Campaign Under Segment",
			UserID:  "1234",
			Segment: "Unicorn",
			Campaigns: []campaignData{
				{
					Platform:  "growth",
					AdAccount: "trinacia",
					Campaign: &entities.Campaign{
						ID:        "11234",
						Budget:    "1bn",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: []string{"11234"},
			Error:    nil,
		},
		{
			Name:    "Multiple Campaigns Under Segment",
			UserID:  "1234",
			Segment: "Trinacia",
			Campaigns: []campaignData{
				{
					Platform:  "growth",
					AdAccount: "trinacia",
					Campaign: &entities.Campaign{
						ID:        "1234",
						Budget:    "500m",
						StartTime: time.Now().String(),
						EndTime:   time.Now().Add(time.Hour * 365).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
				{
					Platform:  "growth",
					AdAccount: "trinacia",
					Campaign: &entities.Campaign{
						ID:        "12345",
						Budget:    "500m",
						StartTime: time.Now().Add(-time.Hour * 24).String(),
						EndTime:   time.Now().Add(time.Hour * 367).String(),
						Targeting: []*genetic.Chromosome{
							{},
						},
						Media: []entities.Media{
							{},
						},
					},
				},
			},
			Expected: []string{"12345", "1234"},
			Error:    nil,
		},
		{
			Name:      "Missing User ID",
			UserID:    "",
			Segment:   "Trinacia",
			Campaigns: nil,
			Expected:  nil,
			Error:     ErrorMissingUserID,
		},
		{
			Name:      "Missing Segment",
			UserID:    "1234",
			Segment:   "",
			Campaigns: nil,
			Expected:  nil,
			Error:     ErrorMissingSegment,
		},
	}
	for _, tc := range cases {
		if tc.Error == nil {
			for _, campaign := range tc.Campaigns {
				testCreateCampaign(t, tc.UserID, campaign.Platform, campaign.AdAccount, tc.Segment, campaign.Campaign)
				defer testDeleteItem(t, "campaigns", campaign.Campaign.ID)
			}

		}
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := New(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			c, err := storage.GetSegmentCampaigns(tc.UserID, tc.Segment)
			assert.Equal(tc.Expected, c)
			assert.Equal(tc.Error, err)
		})
	}
}
