package user

import "bitbucket.org/backend/core/entities"

// Storage interface to get information from database
type Storage interface {
	StoreUser(u *entities.User) error
	GetUser(userID string) (*entities.User, error)
}
