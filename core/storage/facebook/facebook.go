package facebook

import "bitbucket.org/backend/core/entities"

// Storage interface to get information from database
type Storage interface {
	StoreFacebook(userID string, f *entities.Facebook) error
	GetFacebook(userID string) (*entities.Facebook, error)
}
