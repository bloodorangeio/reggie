package reggie

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-resty/resty/v2"
)

const (
	DefaultUserAgent = "reggie/0.3.0 (https://github.com/bloodorangeio/reggie)"
)

// Client is an HTTP(s) client to make requests against an OCI registry.
type (
	Client struct {
		*resty.Client
		Config *clientConfig
	}

	clientConfig struct {
		Address               string
		AuthScope             string
		Username              string
		Password              string
		Debug                 bool
		DefaultName           string
		UserAgent             string
		InsecureSkipTLSVerify bool
	}

	clientOption func(c *clientConfig)
)

// validate the client config.
func (c *clientConfig) validate() error {

	// Check that both Address and UserAgent are defined
	if c.Address == "" {
		return fmt.Errorf("Address is required")
	}

	// Validate the url
	isValid := govalidator.IsURL(c.Address)
	if !isValid {
		return fmt.Errorf("%s is not a valid URL", c.Address)
	}

	if c.UserAgent == "" {
		return fmt.Errorf("UserAgent is required")
	}
	return nil
}

// NewClient builds a new Client from provided options.
func NewClient(address string, opts ...clientOption) (*Client, error) {
	conf := &clientConfig{}
	conf.Address = strings.TrimSuffix(address, "/")
	for _, fn := range opts {
		fn(conf)
	}
	if conf.UserAgent == "" {
		conf.UserAgent = DefaultUserAgent
	}

	// Validate the config
	err := conf.validate()
	if err != nil {
		return nil, err
	}

	client := Client{}
	client.Client = resty.New()
	client.Config = conf
	client.Debug = conf.Debug
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(20))
	client.SetTransport(createTransport(conf.InsecureSkipTLSVerify))

	// TODO: disable this
	// See https://github.com/opencontainers/distribution-spec/issues/396
	// Restly will automatically set Accept based on Content-Type. When a user
	// uses the "SetHeader" method, we intercept it and set the value on the
	// context. If that context value is missing, delete it as it
	// means Resty has automatically set the value for us (bad)
	client.SetPreRequestHook(func(_ *resty.Client, req *http.Request) error {
		acceptHeaderVal := req.Context().Value(contextKeyAcceptHeader)
		if acceptHeaderVal != nil {
			req.Header.Set("Accept", fmt.Sprintf("%s", acceptHeaderVal))
		} else if req.Header.Get("Accept") != "" {
			req.Header.Del("Accept")
		}
		return nil
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

// WithAuthScope overrides the scope provided by the authorization server.
func WithAuthScope(authScope string) clientOption {
	return func(c *clientConfig) {
		c.AuthScope = authScope
	}
}

// WithDefaultName sets the default registry namespace configuration setting.
func WithDefaultName(namespace string) clientOption {
	return func(c *clientConfig) {
		c.DefaultName = namespace
	}
}

// WithDebug enables or disables debug mode.
func WithDebug(debug bool) clientOption {
	return func(c *clientConfig) {
		c.Debug = debug
	}
}

// WithUserAgent overrides the client user agent
func WithUserAgent(userAgent string) clientOption {
	return func(c *clientConfig) {
		c.UserAgent = userAgent
	}
}

// WithInsecureSkipTLSVerify configures the insecure option to skip TLS verification.
func WithInsecureSkipTLSVerify(skip bool) clientOption {
	return func(c *clientConfig) {
		c.InsecureSkipTLSVerify = skip
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
		"<name>":       namespace,
		"<reference>":  r.Reference,
		"<digest>":     r.Digest,
		"<session_id>": r.SessionID,
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
	restyRequest.SetHeader("User-Agent", client.Config.UserAgent)

	return &Request{
		Request:       restyRequest,
		retryCallback: r.RetryCallback,
	}
}

// Do executes a Request and returns a Response.
func (client *Client) Do(req *Request) (*Response, error) {
	resp, err := req.Execute(req.Method, req.URL)
	if err != nil {
		return resp, err
	}
	if resp.IsUnauthorized() {
		resp, err = client.retryRequestWithAuth(req, resp)
	}
	return resp, err
}

// adapted from Resty: https://github.com/go-resty/resty/blob/de0735f66dae7abf8fb1073b4ace3032c1491424/client.go#L928
func createTransport(insecureSkipTLSVerify bool) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		DisableCompression:    true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipTLSVerify, //nolint: gosec
		},
	}
}
