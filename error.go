package reggie

import (
	spec "github.com/opencontainers/distribution-spec/specs-go/v1"
)

type (
	// ErrorInfo describes a server error returned from a registry.
	ErrorInfo struct {
		*spec.ErrorInfo
	}
)
