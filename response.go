package reggie

import (
	"gopkg.in/resty.v1"
)

type (
	// Response is an HTTP response returned from an OCI registry.
	Response struct {
		*resty.Response
	}
)
