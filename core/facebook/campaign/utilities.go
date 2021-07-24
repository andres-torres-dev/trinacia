package campaign

import (
	"crypto/rand"

	"bitbucket.org/backend/core/logger"
)

// randomName is used to securilly generating unique names for adsets and ads
func randomName() (string, error) {
	b := make([]byte, 15)
	_, err := rand.Read(b)
	if err != nil {
		return "", &logger.Error{
			Level:   "Panic",
			Message: "Unable to generate random name.",
			Err:     err,
		}
	}

	return string(b), nil
}
