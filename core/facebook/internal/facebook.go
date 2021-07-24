package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

const (
	graphAPI     = "graph.facebook.com"
	graphVersion = "v8.0"
)

// FacebookPaging used to page requests
type FacebookPaging struct {
	Cursors struct {
		Before string `json:"before,omitempty"`
		After  string `json:"after,omitempty"`
	} `json:"cursors,omitempty"`
	Next string `json:"next,omitempty"`
}

// FacebookError represent an error encountered while making a request
// to the graph api
type FacebookError struct {
	Title   string `json:"error_user_title,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	Code    int    `json:"code,omitempty"`
	SubCode int    `json:"error_subcode,omitempty"`
}

func (e *FacebookError) Error() string {
	s, err := json.Marshal(e)
	if err != nil {
		log.Fatal(err)
	}
	return string(s)
}

// SetURL relative url and query to request platform
func SetURL(relative string, query url.Values) string {
	u := url.URL{}
	u.Scheme = "https"
	u.Host = graphAPI
	u.Path = fmt.Sprintf("%s/%s", graphVersion, relative)
	u.RawQuery = query.Encode()

	return u.String()
}
