package twofa

import (
	"errors"
)

var ErrCanNotCreate2Factor = errors.New("error during the 2FA creation")
var ErrInvalidCode = errors.New("the code entered is invalid")
