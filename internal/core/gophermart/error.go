package gophermart

import "errors"

var (
	ErrPasswordNotValid    = errors.New("password is not valid")
	ErrLoginNotValid       = errors.New("login is not valid")
	ErrPasswordNotEquale   = errors.New("password not equale")
	ErrOrderNumberNotValid = errors.New("order number not valid")
)
