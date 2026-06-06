package main

import "errors"

var (
	ErrMissingClientID       = errors.New("client ID is required")
	ErrMissingClientSecret   = errors.New("client secret is required")
	ErrMissingRedirectURI    = errors.New("redirect URI is required")
	ErrMissingAuthEndpoint   = errors.New("authorization endpoint is required")
	ErrMissingTokenEndpoint  = errors.New("token endpoint is required")
	ErrMissingStateSecret    = errors.New("state secret is required")
	ErrInvalidState          = errors.New("invalid state parameter")
	ErrInvalidCode           = errors.New("invalid authorization code")
	ErrInvalidGrant          = errors.New("invalid grant")
	ErrTokenExpired          = errors.New("token has expired")
	ErrInvalidToken          = errors.New("invalid token")
	ErrLLMRequestFailed      = errors.New("failed to request LLM")
)
