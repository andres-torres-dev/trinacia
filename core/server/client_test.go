package server

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)
	t.Run("Error Nil URL", func(t *testing.T) {
		client := New(&flag{})
		resp, err := client.Get("")
		assert.Equal(ErrorNilURL, err)
		assert.Equal((*http.Response)(nil), resp)

		resp, err = client.Post("", nil)
		assert.Equal(ErrorNilURL, err)
		assert.Equal((*http.Response)(nil), resp)
	})

	t.Run("Error Set Request in methods", func(t *testing.T) {
		client := New(&flag{
			requestError: true,
		})
		resp, err := client.Get("test.com")
		assert.Equal((*http.Response)(nil), resp)
		assert.Error(err)

		resp, err = client.Post("test.com", nil)
		assert.Equal((*http.Response)(nil), resp)
		assert.Error(err)
	})

	t.Run("Set Request", func(t *testing.T) {
		client := New(&flag{})
		req, err := client.SetRequest("GET", "https://www.google.com", nil)
		if err != nil {
			t.Error("unable to set client request error: ", err)
		}
		assert.Equal("GET", req.Method)
		u, err := url.Parse("https://www.google.com")
		if err != nil {
			t.Fatal("unable to parse url testing Set Request")
		}
		assert.Equal(u, req.URL)
	})

	t.Run("GET", func(t *testing.T) {
		client := New(&flag{})
		resp, err := client.Get("https://www.google.com")
		if err != nil {
			t.Error("GET request failing error: ", err)
		}
		err = resp.Body.Close()
		if err != nil {
			t.Fatal("unable to close body in get request")
		}
		if resp.StatusCode != 200 {
			t.Error("Unexpected Response status code")
		}
	})

	t.Run("POST", func(t *testing.T) {
		client := New(&flag{})
		resp, err := client.Post("https://www.google.com", nil)
		_ = resp.Body.Close()
		if err != nil {
			t.Error("POST request failing error: ", err)
		}
		if resp.StatusCode != 405 {
			t.Error("Unexpected Response status code")
		}
	})

	t.Run("Error Do", func(t *testing.T) {
		client := New(&flag{
			doError: true,
		})
		req, err := client.SetRequest("GET", "google.com", nil)
		if err != nil {
			t.Fatal("Unexpected error setting client request")
		}
		resp, err := client.Do(req)
		assert.Equal((*http.Response)(nil), resp)
		assert.Error(err)
	})
}
