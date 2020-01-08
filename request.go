package reggie

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"

	"gopkg.in/resty.v1"
)

type (
	// Request is an HTTP request to be sent to an OCI registry.
	Request struct {
		*resty.Request
	}

	requestConfig struct {
		Name      string
		Reference string
		Digest    string
		SessionID string
	}

	requestOption func(c *requestConfig)
)

// WithName sets the namespace per a single request.
func WithName(name string) requestOption {
	return func(c *requestConfig) {
		c.Name = name
	}
}

// WithReference sets the reference per a single request.
func WithReference(ref string) requestOption {
	return func(c *requestConfig) {
		c.Reference = ref
	}
}

// WithDigest sets the digest per a single request.
func WithDigest(digest string) requestOption {
	return func(c *requestConfig) {
		c.Digest = digest
	}
}

// WithSessionID sets the session ID per a single request.
func WithSessionID(id string) requestOption {
	return func(c *requestConfig) {
		c.SessionID = id
	}
}

func (req *Request) SetHeader(header, content string) *Request {
	req.Request.SetHeader(header, content)
	return req
}

func (req *Request) SetQueryParam(param, content string) *Request {
	req.Request.SetQueryParam(param, content)
	return req
}

// Execute validates a Request and executes it.
func (req *Request) Execute(method, url string) (*Response, error) {
	err := validateRequest(req)
	if err != nil {
		return nil, err
	}

	restyResponse, err := req.Request.Execute(method, url)
	if err != nil {
		return nil, err
	}

	resp := &Response{restyResponse}
	return resp, err
}

func (req *Request) SetBody(body []byte) *Request {
	req.Request.SetBody(ioutil.NopCloser(bytes.NewReader(body)))
	return req
}

func validateRequest(req *Request) error {
	re := regexp.MustCompile("<name>|<reference>|<digest>|<session_id>|//{2,}")
	matches := re.FindAllString(req.URL, -1)
	if len(matches) == 0 {
		return nil
	}
	return fmt.Errorf("request is invalid")
}
