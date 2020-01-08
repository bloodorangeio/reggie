package reggie

import (
	"encoding/json"
	"gopkg.in/resty.v1"
	"net/url"
)

type (
	// Response is an HTTP response returned from an OCI registry.
	Response struct {
		*resty.Response
	}
)

// GetRelativeLocation returns the path component of the URL contained
// in the `Location` header of the response.
func (resp *Response) GetRelativeLocation() string {
	loc := resp.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		return ""
	}

	path := u.Path
	if q := u.RawQuery; q != "" {
		path += "?" + q
	}

	return path
}

// GetAbsoluteLocation returns the full URL, including protocol and host,
// of the location contained in the `Location` header of the response.
func (resp *Response) GetAbsoluteLocation() string {
	return resp.Header().Get("Location")
}

func (resp *Response) Errors() (*Error, error) {
	e := &Error{}
	bodyBytes := []byte(resp.String())
	err := json.Unmarshal(bodyBytes, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}
