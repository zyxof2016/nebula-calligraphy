package service

import "errors"

var ErrPersistence = errors.New("persistence operation failed")
var ErrUnauthorized = errors.New("unauthorized")
var ErrIdentityUnavailable = errors.New("identity service unavailable")
