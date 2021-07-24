package user

import (
	"errors"
	"time"

	"bitbucket.org/backend/core/entities"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type dynamo struct {
	svc *dynamodb.DynamoDB
}

// New isntanciates a session dynamo session to query information
// about users
func New(sess *session.Session) Storage {
	d := &dynamo{
		svc: dynamodb.New(sess),
	}

	return d
}

const (
	// TableName is the table used to store the data
	TableName = "trinacia"
)

var (
	// ErrorUserAlreadyExists returned when a new user is updating an already existing user
	ErrorUserAlreadyExists = errors.New("the user can't be created because it already exists")
	// ErrorInvalidUser user not found
	ErrorInvalidUser = errors.New("Invalid User ID")
	// ErrorMissingUserID missing user ID
	ErrorMissingUserID = errors.New("Missing User ID")
	// ErrorMissingUser nil pointer reference to User
	ErrorMissingUser = errors.New("Nil pointer reference passed as User")
)

func (d *dynamo) StoreUser(u *entities.User) error {
	if u == nil {
		return ErrorMissingUser
	}
	if u.ID == "" {
		return ErrorMissingUserID
	}
	if us, err := d.GetUser(u.ID); us != nil || err != ErrorInvalidUser && err != nil {
		switch {
		case us != nil:
			return ErrorUserAlreadyExists
		default:
			return err
		}
	}

	var creationTime = time.Now().String()
	u.CreationTime = creationTime

	in := &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String("users"),
			},
			"key": {
				S: aws.String(u.ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#id":            aws.String("id"),
			"#name":          aws.String("name"),
			"#email":         aws.String("email"),
			"#creation_time": aws.String("creation_time"),
			"#sort":          aws.String("sort"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(u.ID),
			},
			":name": {
				S: aws.String(u.Name),
			},
			":email": {
				S: aws.String(u.Email),
			},
			":creation_time": {
				S: aws.String(creationTime),
			},
		},
		UpdateExpression: aws.String("set #id=:id, #name=:name, #email=:email, #sort=:creation_time, #creation_time=:creation_time"),
	}
	_, err := d.svc.UpdateItem(in)

	return err
}

func (d *dynamo) GetUser(userID string) (*entities.User, error) {
	if userID == "" {
		return nil, ErrorMissingUserID
	}
	u := &entities.User{}

	in := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"partition": {
				S: aws.String("users"),
			},
			"key": {
				S: aws.String(userID),
			},
		},
	}
	out, err := d.svc.GetItem(in)
	if err != nil {
		return nil, err
	}
	err = dynamodbattribute.UnmarshalMap(out.Item, u)
	if err != nil {
		return nil, err
	}

	if u.ID == "" {
		return nil, ErrorInvalidUser
	}

	return u, nil
}
