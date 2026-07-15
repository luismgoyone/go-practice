package service

import (
	"errors"

	"github.com/luismgoyone/go-practice/internal/repository"
)

// ValidationError is a business-rule failure (HTTP 400).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// IsNotFound reports repository.ErrNotFound without handlers importing mongo.
func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}
