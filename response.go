package reggie

import (
	"net/http"

	"gopkg.in/resty.v1"
)

type (
	Response struct {
		*resty.Response
	}
)

func (resp *Response) IsUnauthorized() bool {
	return resp.StatusCode() == http.StatusUnauthorized
}
