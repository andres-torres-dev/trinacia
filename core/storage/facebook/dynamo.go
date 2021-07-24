package facebook

import (
	"errors"

	"bitbucket.org/backend/core/entities"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type dynamo struct {
	svc *dynamodb.DynamoDB
}

// NewFacebook isntanciates a session dynamo session to query information
// about users' Facebook accounts
func NewFacebook(sess *session.Session) Storage {
	return &dynamo{
		svc: dynamodb.New(sess),
	}
}

const (
	// TableName is the table used to store the data
	TableName = "trinacia"
)

var (
	// ErrorMissingUserID missing user ID
	ErrorMissingUserID = errors.New("Missing User ID")
	// ErrorMissingFacebook nil pointer reference to facebook
	ErrorMissingFacebook = errors.New("Nil pointer reference passed as Facebook entitie")
	// ErrorMissingFacebookAccessToken missing facebook access token
	ErrorMissingFacebookAccessToken = errors.New("Missing facebook access token")
)

func (d *dynamo) StoreFacebook(userID string, f *entities.Facebook) error {
	if userID == "" {
		return ErrorMissingUserID
	}
	if f == nil {
		return ErrorMissingFacebook
	}
	if f.AccessToken == "" {
		return ErrorMissingFacebookAccessToken
	}

	pages, err := dynamodbattribute.Marshal(f.Pages)
	if err != nil {
		return err
	}
	adAccounts, err := dynamodbattribute.Marshal(f.AdAccounts)
	if err != nil {
		return err
	}

	in := &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(userID),
			},
			"key": {
				S: aws.String("facebook"),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#accessToken": aws.String("access_token"),
			"#pages":       aws.String("pages"),
			"#adAccounts":  aws.String("ad_accounts"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":accessToken": {
				S: aws.String(f.AccessToken),
			},
			":pages":      pages,
			":adAccounts": adAccounts,
		},
		UpdateExpression: aws.String("set #accessToken=:accessToken, #pages=:pages, #adAccounts=:adAccounts"),
	}
	_, err = d.svc.UpdateItem(in)
	if err != nil {
		return err
	}

	return nil
}

func (d *dynamo) GetFacebook(userID string) (*entities.Facebook, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	f := &entities.Facebook{}

	in := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String(userID),
			},
			"key": {
				S: aws.String("facebook"),
			},
		},
	}
	out, err := d.svc.GetItem(in)
	if err != nil {
		return nil, err
	}
	err = dynamodbattribute.UnmarshalMap(out.Item, f)
	if err != nil {
		return nil, err
	}

	return f, nil
}
