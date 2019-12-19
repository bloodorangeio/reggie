package reggie

import "gopkg.in/resty.v1"

const (
	GET = resty.MethodGet
	PUT = resty.MethodPut
	PATCH = resty.MethodPatch
	DELETE = resty.MethodDelete
	POST = resty.MethodPost
	HEAD = resty.MethodHead
	OPTIONS = resty.MethodOptions
)
