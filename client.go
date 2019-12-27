package reggie

import (
	"fmt"
	"net/http"
	"strings"

	reg "github.com/genuinetools/reg/registry"
	"gopkg.in/resty.v1"
)

type (
	// Client is an HTTP(s) client to make requests against an OCI registry.
	Client struct {
		*resty.Client
		Config *clientConfig
	}

	clientConfig struct {
		Address     string
		Username    string
		Password    string
		Debug       bool
		DefaultName string
	}

	clientOption func(c *clientConfig)
)

// NewClient builds a new Client from provided options.
func NewClient(address string, opts ...clientOption) (*Client, error) {
	conf := &clientConfig{}
	conf.Address = strings.TrimSuffix(address, "/")
	for _, fn := range opts {
		fn(conf)
	}

	// TODO: validate config here, return error if it aint no good

	client := Client{}
	client.Client = resty.New()
	client.Config = conf

	// For client transport, use reg's multilayer RoundTripper for "Docker-style" auth
	client.SetTransport(&reg.BasicTransport{
		Transport: &reg.TokenTransport{
			Transport: http.DefaultTransport,
			Username:  client.Config.Username,
			Password:  client.Config.Password,
		},
		URL:      client.Config.Address,
		Username: client.Config.Username,
		Password: client.Config.Password,
	})

	return &client, nil
}

// WithUsernamePassword sets registry username and password configuration settings.
func WithUsernamePassword(username string, password string) clientOption {
	return func(c *clientConfig) {
		c.Username = username
		c.Password = password
	}
}

// WithDefaultName sets the default registry namespace configuration setting.
func WithDefaultName(namespace string) clientOption {
	return func(c *clientConfig) {
		c.DefaultName = namespace
	}
}

func WithDebug(debug bool) clientOption {
	return func(c *clientConfig) {
		c.Debug = debug
	}
}

// SetDefaultName sets the default registry namespace to use for building a Request.
func (client *Client) SetDefaultName(namespace string) {
	client.Config.DefaultName = namespace
}

// NewRequest builds a new Request from provided options.
func (client *Client) NewRequest(method string, path string, opts ...requestOption) *Request {
	restyRequest := client.Client.NewRequest()
	restyRequest.Method = method
	r := &requestConfig{}
	for _, o := range opts {
		o(r)
	}

	namespace := client.Config.DefaultName
	if r.Name != "" {
		namespace = r.Name
	}

	replacements := map[string]string{
		":name":    namespace,
		":ref":     r.Reference,
		":digest":  r.Digest,
		":session": r.SessionID,
	}

	// substitute known path params
	for k, v := range replacements {
		if v != "" {
			path = strings.Replace(path, k, v, -1)
		}
	}

	path = strings.TrimPrefix(path, "/")

	url := fmt.Sprintf("%s/%s", client.Config.Address, path)
	restyRequest.URL = url

	return &Request{restyRequest}
}

// Do executes a Request and returns a Response.
func (client *Client) Do(req *Request) (*Response, error) {
	return req.Execute(req.Method, req.URL)
}
