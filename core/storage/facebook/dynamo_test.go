package facebook

import (
	"testing"

	"bitbucket.org/backend/core/entities"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func testCreateFacebook(t *testing.T, userID string, f *entities.Facebook) {
	t.Helper()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	storage := NewFacebook(sess)

	err = storage.StoreFacebook(userID, f)
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

func TestStoreFacebook(t *testing.T) {
	cases := []struct {
		Name     string
		UserID   string
		Facebook *entities.Facebook
		Error    error
	}{
		{
			Name:   "Correct",
			UserID: "1234",
			Facebook: &entities.Facebook{
				Pages: []entities.Page{
					{
						Category: "Unicorn",
						Name:     "Trinacia",
						ID:       "1234",
						Instagram: []entities.Instagram{
							{
								ID:   "1234",
								Name: "Trinacia",
							},
						},
						AccessToken: "1234",
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "act_1234",
						ID:        "1234",
						Name:      "Trinacia",
					},
				},
				AccessToken: "1234",
			},
			Error: nil,
		},
		{
			Name:   "Missing User ID",
			UserID: "",
			Facebook: &entities.Facebook{
				Pages: []entities.Page{
					{
						Category: "Unicorn",
						Name:     "Trinacia",
						ID:       "1234",
						Instagram: []entities.Instagram{
							{
								ID:   "1234",
								Name: "Trinacia",
							},
						},
						AccessToken: "1234",
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "act_1234",
						ID:        "1234",
						Name:      "Trinacia",
					},
				},
				AccessToken: "1234",
			},
			Error: ErrorMissingUserID,
		},
		{
			Name:   "Missing Access Token",
			UserID: "1234",
			Facebook: &entities.Facebook{
				Pages: []entities.Page{
					{
						Category: "Unicorn",
						Name:     "Trinacia",
						ID:       "1234",
						Instagram: []entities.Instagram{
							{
								ID:   "1234",
								Name: "Trinacia",
							},
						},
						AccessToken: "1234",
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "act_1234",
						ID:        "1234",
						Name:      "Trinacia",
					},
				},
			},
			Error: ErrorMissingFacebookAccessToken,
		},
		{
			Name:     "Nil Pointer Reference",
			UserID:   "1234",
			Facebook: nil,
			Error:    ErrorMissingFacebook,
		},
	}
	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := NewFacebook(sess)

	for _, tc := range cases {
		if tc.Error == nil {
			defer testDeleteItem(t, tc.UserID, "facebook")
		}
		t.Run(tc.Name, func(t *testing.T) {
			err := storage.StoreFacebook(tc.UserID, tc.Facebook)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetFacebook(t *testing.T) {
	cases := []struct {
		Name     string
		UserID   string
		Expected *entities.Facebook
		Create   bool
		Error    error
	}{
		{
			Name:   "Correct",
			UserID: "1234",
			Expected: &entities.Facebook{
				Pages: []entities.Page{
					{
						Category: "Unicorn",
						Name:     "Trinacia",
						ID:       "1234",
						Instagram: []entities.Instagram{
							{
								ID:   "1234",
								Name: "Trinacia",
							},
						},
						AccessToken: "1234",
					},
				},
				AdAccounts: []entities.AdAccount{
					{
						AccountID: "act_1234",
						ID:        "1234",
						Name:      "Trinacia",
					},
				},
				AccessToken: "1234",
			},
			Create: true,
			Error:  nil,
		},
		{
			Name:     "Missing User ID",
			UserID:   "",
			Expected: nil,
			Error:    ErrorMissingUserID,
		},
		{
			Name:     "No Data Found",
			UserID:   "123412341234123",
			Expected: &entities.Facebook{},
			Error:    nil,
		},
	}

	// create test facebook data
	for _, tc := range cases {
		if tc.Create {
			testCreateFacebook(t, tc.UserID, tc.Expected)
			defer testDeleteItem(t, tc.UserID, "facebook")
		}
	}

	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	storage := NewFacebook(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			f, err := storage.GetFacebook(tc.UserID)
			assert.Equal(tc.Expected, f)
			assert.Equal(tc.Error, err)
		})
	}
}
