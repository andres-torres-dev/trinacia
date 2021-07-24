package user

import (
	"testing"

	"bitbucket.org/backend/core/entities"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func testCreateUser(t *testing.T, user *entities.User) {
	t.Helper()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	storage := New(sess)

	err = storage.StoreUser(user)
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

func TestStoreUser(t *testing.T) {
	cases := []struct {
		Name  string
		User  *entities.User
		Error error
	}{
		{
			Name: "New User",
			User: &entities.User{
				ID:    "1234",
				Name:  "Andres",
				Email: "andres@trinacia.com",
			},
			Error: nil,
		},
		{
			Name: "User Already Exists",
			User: &entities.User{
				ID:    "1234",
				Name:  "Andres",
				Email: "andres@trinacia.com",
			},
			Error: ErrorUserAlreadyExists,
		},
		{
			Name: "Missing User ID",
			User: &entities.User{
				ID:    "",
				Name:  "Andres",
				Email: "andres@trinacia.com",
			},
			Error: ErrorMissingUserID,
		},
		{
			Name:  "Nil Pointer Reference",
			User:  nil,
			Error: ErrorMissingUser,
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
			defer testDeleteItem(t, "users", tc.User.ID)
		}
		t.Run(tc.Name, func(t *testing.T) {
			err := storage.StoreUser(tc.User)
			assert.Equal(tc.Error, err)
		})
	}
}

func TestGetUser(t *testing.T) {
	cases := []struct {
		Name     string
		ID       string
		Expected *entities.User
		Error    error
	}{
		{
			Name: "Existing User",
			ID:   "1234",
			Expected: &entities.User{
				ID:    "1234",
				Name:  "Andres",
				Email: "andres@trinacia.com",
			},
			Error: nil,
		},
		{
			Name:     "Invalid User",
			ID:       "12345",
			Expected: nil,
			Error:    ErrorInvalidUser,
		},
		{
			Name:     "Missing User ID",
			ID:       "",
			Expected: nil,
			Error:    ErrorMissingUserID,
		},
	}
	assert := assert.New(t)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// create test users
	for _, tc := range cases {
		if tc.Error == nil {
			testCreateUser(t, tc.Expected)
			defer testDeleteItem(t, "users", tc.ID)
		}
	}

	storage := New(sess)

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			user, err := storage.GetUser(tc.ID)
			assert.Equal(tc.Expected, user)
			assert.Equal(tc.Error, err)
		})
	}
}
