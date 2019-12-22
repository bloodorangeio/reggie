package reggie

import (
	"regexp"

	"gopkg.in/resty.v1"
)

type (
	Request struct {
		*resty.Request
	}
)

func (req *Request) Execute(method, url string) (*Response, error) {
	restyResponse, err := req.Request.Execute(method, url)
	if err != nil {
		return nil, err
	}
	resp := &Response{restyResponse}
	return resp, err
}

func (req *Request) isValid() bool {
	re := regexp.MustCompile(":name|:digest|:uuid|//{2,}")
	matches := re.FindAllString(req.URL, -1)
	if len(matches) == 0 {
		return true
	}
	return false
}

func (req *Request) deleteQueryParams() {
	for k, _ := range req.QueryParam {
		req.QueryParam.Del(k)
	}
}
