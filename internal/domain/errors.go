package domain

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("already exists")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrDevicePending    = errors.New("device is pending approval")
	ErrDeviceRejected   = errors.New("device has been rejected")
	ErrInvalidInput     = errors.New("invalid input")
	ErrArtifactInUse    = errors.New("artifact is referenced by active deployments")
	ErrDeploymentActive = errors.New("deployment is already active")
)
