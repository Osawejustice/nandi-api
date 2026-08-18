package services

import "errors"

var (
	ErrUnavailable        = errors.New("service unavailable")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrSlugTaken          = errors.New("organization slug already taken")
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation failed")
	ErrTenantRequired     = errors.New("tenant slug is required")
	ErrTenantSuspended    = errors.New("tenant is suspended")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrProvider           = errors.New("provider error")
	ErrUnsupportedChannel = errors.New("unsupported channel")
	ErrInvalidState       = errors.New("invalid conversation state")
)
