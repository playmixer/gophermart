package errstore

import "errors"

var (
	ErrLoginNotUnique             = errors.New("login not unique")
	ErrNotFoundData               = errors.New("not found data")
	ErrOrderWasCreatedAnotherUser = errors.New("order was created another user")
	ErrOrderWasCreatedByUser      = errors.New("order was create by user")
	ErrBalansNotEnough            = errors.New("balance is not enough")
)
