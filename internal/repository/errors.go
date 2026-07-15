package repository

import "errors"

// ErrNotFound means no document matched the query.
// Handlers map this to HTTP 404.
var ErrNotFound = errors.New("not found")
