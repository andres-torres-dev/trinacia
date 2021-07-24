package auth

import (
	"strings"

	"bitbucket.org/backend/core/entities"
	"bitbucket.org/backend/core/logger"
	"bitbucket.org/backend/core/storage/user"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

// Auth interface for user authentication with Trinacia
type Auth interface {
	GetUser(authentication string) (*entities.User, error)
}

type cognito struct {
	svc     *cognitoidentityprovider.CognitoIdentityProvider
	storage user.Storage
}

// NewCognitoAuth instanciates a cognito service
func NewCognitoAuth(sess *session.Session) Auth {
	return &cognito{
		svc:     cognitoidentityprovider.New(sess),
		storage: user.New(sess),
	}
}

// GetUser checks the authorization header and retrieves the user token
func (c *cognito) GetUser(authentication string) (*entities.User, error) {
	if !strings.Contains(authentication, "Bearer ") {
		return nil, &logger.Error{
			Level:   "Error",
			Message: "Incorrect Authorization Header",
		}
	}

	token := strings.Split(authentication, "Bearer ")[1]
	in := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(token),
	}
	out, err := c.svc.GetUser(in)
	if err != nil {
		return nil, &logger.Error{
			Level:   "Error",
			Message: "Encounter error while retrieving cognito user information",
			Err:     err,
		}
	}

	user, err := c.storage.GetUser(*out.Username)
	if err != nil {
		return nil, err
	}

	return user, nil
}
