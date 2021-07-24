package internal

import (
	"fmt"
	"net/url"
	"testing"

	"bitbucket.org/backend/core/server"
	"bitbucket.org/backend/core/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/stretchr/testify/assert"
)

func TestHelpers(t *testing.T) {
	assert := assert.New(t)

	t.Run("Set URL", func(t *testing.T) {
		uV := url.Values{}
		uV.Add("access_token", "1234")
		uV.Add("fields", "id,name,category,access_token")
		u := setURL(fmt.Sprintf("%s/accounts", "1234"), uV)

		expected := "https://graph.facebook.com/v8.0/1234/accounts?access_token=1234&fields=id%2Cname%2Ccategory%2Caccess_token"

		assert.Equal(expected, u)
	})

	t.Run("New Facebook", func(t *testing.T) {
		c := server.New()
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-west-2"),
		})
		if err != nil {
			t.Fatal("Unable to start session: ", err)
		}
		s := storage.NewDynamo(sess)
		f := New(c, s)
		assert.Equal(&facebook{}, f)
		assert.NotNil(client, store)
	})

	t.Run("Facebook Error", func(t *testing.T) {
		f := &facebookError{
			Message: "Invalid OAuth access token.",
			Type:    "OAuthException",
			Code:    190,
		}
		e := `{"message":"Invalid OAuth access token.","type":"OAuthException","code":190}`
		assert.Equal(e, f.Error())
	})
}
