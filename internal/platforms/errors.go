package platforms

import "errors"

var (
	ErrUnknownPlatform = errors.New("unknown platform")
	ErrPlanned         = errors.New("platform not implemented")
	ErrUsage           = errors.New("usage")
	ErrMissingTool     = errors.New("required tool not found")
	ErrOffline         = errors.New("offline setup incomplete")
)
