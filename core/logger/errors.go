package logger

import (
	"encoding/json"
	"log"
)

type Error struct {
	Level         string      `json:"level"`
	Err           error       `json:"error"`
	Context       interface{} `json:"context"`
	User          string      `json:"user"`
	Message       string      `json:"message"`
	ClientMessage string      `json:"client_message"`
}

func (e *Error) Error() string {
	rerr := struct {
		Level         string      `json:"level"`
		Err           string      `json:"error"`
		Context       interface{} `json:"context"`
		User          string      `json:"user"`
		Message       string      `json:"message"`
		ClientMessage string      `json:"client_message"`
	}{
		Level:         e.Level,
		Context:       e.Context,
		User:          e.User,
		Message:       e.Message,
		ClientMessage: e.ClientMessage,
	}
	if e.Err != nil {
		rerr.Err = e.Err.Error()
	}
	s, err := json.Marshal(rerr)
	if err != nil {
		log.Fatal("unable to marshal error: ", err)
	}

	return string(s)
}
