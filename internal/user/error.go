package user

import (
	"errors"
	"fmt"
)

var ErrFirstNameRequired = errors.New("first name is required")
var ErrLastNameRequired = errors.New("last name is required")
var ErrUsernameRequired = errors.New("username is required")
var ErrPasswordRequired = errors.New("password is required")
var ErrCodeRequired = errors.New("code is required")

type ErrNotFound struct {
	UserID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("user '%s' doesn't exist", e.UserID)
}
