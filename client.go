package reggie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/resty.v1"
)

type (
	Request struct {
		*resty.Request
	}

	Response struct {
		*resty.Response
	}

	Client struct {
		*resty.Client
		Config struct {
			Address   string
			Namespace string
			Auth      struct {
				Basic struct {
					Username string
					Password string
				}
			}
		}
	}

	authHeader struct {
		Realm   string
		Service string
		Scope   string
	}

	authInfo struct {
		Token string `json:"token"`
	}
)

func (resp *Response) IsUnauthorized() bool {
	return resp.StatusCode() == http.StatusUnauthorized
}

func (client *Client) NewRequest(method, path string) *Request {
	restyRequest := client.Client.NewRequest()
	restyRequest.Method = method
	path = strings.Replace(path, ":namespace", client.Config.Namespace, -1)
	url := fmt.Sprintf("%s%s", client.Config.Address, path)
	restyRequest.URL = url
	return &Request{restyRequest}
}

func (req *Request) Execute(method, url string) (*Response, error) {
	restyResponse, err := req.Request.Execute(method, url)
	if err != nil {
		return nil, err
	}
	resp := &Response{restyResponse}
	return resp, err
}

func (client *Client) Do(req *Request) (*Response, error) {
	resp, err := req.Execute(req.Method, req.URL)
	if err != nil {
		return nil, err
	}

	if resp.IsUnauthorized() {
		resp, err = client.retryRequestWithAuth(req, resp)
	}

	return resp, err
}

func (client *Client) retryRequestWithAuth(originalRequest *Request, originalResponse *Response) (*Response, error) {
	authHeaderRaw := originalResponse.Header().Get("Www-Authenticate")
	if authHeaderRaw == "" {
		return originalResponse, nil
	}

	h := parseAuthHeader(authHeaderRaw)

	req := resty.R()
	req.SetQueryParam("service", h.Service)
	req.SetQueryParam("scope", h.Scope)
	req.SetHeader("Accept", "application/json")
	req.SetBasicAuth(client.Config.Auth.Basic.Username, client.Config.Auth.Basic.Password)
	authResp, err := req.Execute(resty.MethodGet, h.Realm)
	if err != nil {
		return nil, err
	}

	var info authInfo
	bodyBytes := authResp.Body()
	err = json.Unmarshal(bodyBytes, &info)
	if err != nil {
		return nil, err
	}

	originalRequest.SetAuthToken(info.Token)
	return originalRequest.Execute(originalRequest.Method, originalRequest.URL)
}

func parseAuthHeader(authHeaderRaw string) *authHeader {
	re := regexp.MustCompile(`([a-zA-z]+)="(.+?)"`)
	matches := re.FindAllStringSubmatch(authHeaderRaw, -1)
	m := make(map[string]string)
	for i := 0; i < len(matches); i++ {
		m[matches[i][1]] = matches[i][2]
	}
	var h authHeader
	mapstructure.Decode(m, &h)
	return &h
}
