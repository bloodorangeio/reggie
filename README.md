# reggie

[![GitHub Actions status](https://github.com/bloodorangeio/reggie/workflows/build/badge.svg)](https://github.com/bloodorangeio/reggie/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/bloodorangeio/reggie)](https://goreportcard.com/report/github.com/bloodorangeio/reggie)
[![GoDoc](https://godoc.org/github.com/bloodorangeio/reggie?status.svg)](https://godoc.org/github.com/bloodorangeio/reggie)

Simple Go HTTP client for OCI distribution, built on top of 
[Resty](https://github.com/go-resty/resty) and [reg](https://github.com/genuinetools/reg).

## Primary Components

### Client (struct)

`Client` is a struct that represents an HTTP client, along with its configuration:

```go
type (
    Client struct {
        *resty.Client
        Config *clientConfig
    }

    clientConfig struct {
        Address     string
        Username    string
        Password    string
        DefaultName string
    }
)
```

`Client`s are intended to be constructed using the `NewClient` function:

```go
func NewClient(address string, opts ...clientOption) (*Client, error)
```

An example follows:
```go
client, err := NewClient("https://quay.io")
```

Optionally, `NewClient` accepts any of two optional function arguments in any combination, exemplified here:

##### WithUsernamePassword (function)
reggie handles authorization and authentication automatically and implicitly.  Simply supply the
optional `WithUsernamePassword` function as a parameter to `NewClient`.

```go
func WithUsernamePassword(username, password string) clientOption
```
```go
client, err := NewClient("https://quay.io", WithUsernamePassword("username", "password"))
```

##### WithDefaultName (function)
`WithDefaultName` is a function taking a `namespace` `string` argument. This namespace will be automatically substituted
for the special string `:name` in requests, unless the namespace is overridden by an individual request.

```go
func WithDefaultName(namespace string) clientOption
```
```go
client, err := NewClient("https://quay.io", WithDefaultName("my/own/repository"))
```

A Request is created like so:
```go
client, err := NewClient("https://quay.io", WithDefaultName("my/own/repository"))
req := client.NewRequest(reggie.GET "/v2/:name/tags/list")
```

Here, `req` becomes a `GET` request to `https://quay.io/v2/my/own/repository/tags/list`

### Request (struct)

`Request` is a struct that represents an HTTP Request, wrapped around resty's Request struct:

```go
Request struct {
    *resty.Request
}
```

`Request`s are intended to be created by the Client:
```go
client, err := reggie.NewClient("https://quay.io")
if err != nil {
    panic(err)
}
req := client.NewRequest(reggie.GET, "/v2/")
```

The Client struct's `NewRequest` function supports four possible optional argument functions (in any combination), 
each exemplified here:
```go
req := client.NewRequest(reggie.GET, "/v2/:name/tags/list", 
    WithName("my/other/repo")) // "/v2/my/other/repo/tags/list"

req = client.NewRequest(reggie.PUT, "/v2/my/repo/manifests/:ref",
	WithRef("test1.0")) // "/v2/my/repo/manifests/test1.0"

req = client.NewRequest(reggie.HEAD, "/v2/my/repo/blobs/:digest", 
    WithDigest(<some-digest>)) // "/v2/my/repo/blobs/<some-digest>"

req = client.NewRequest(reggie.PUT, "/v2/my/repo/blobs/uploads/:session", 
    reggie.WithSessionID(<some-session-id>)) // "/v2/my/repo/blobs/uploads/<some-session-id>"
```

### Response (struct)

`Response` is a struct that represents an HTTP response, wrapped around resty's Response struct:
```go
type Response struct {
    *resty.Response
}
```

With reggie, `Response`s are typically created as a product of the `Client`'s `Do` function:
```go
func (client *Client) Do(req *Request) (*Response, error)
```
```go
client, err := NewClient("https://quay.io", WithDefaultName("my/repo"))
reqest := client.NewRequest(reggie.GET, '/v2/:name/tags/list')
response, err := client.Do(request)
```

## Usage

### Path substitutions
reggie uses a domain-specific language to supply various parts of the URI path.  One of these, `:name`, can be set at
the client level to be used for all requests made by that client, OR at the request level for individual requests.
Below is a table of the possible substitutions:


| Special string | Description                                       | Required function parameter                            |
|----------------|---------------------------------------------------|--------------------------------------------------------|
| `:name`        | The namespace of a repository within the registry | `WithDefaultName` (`Client`) or `WithName` (`Request`) |
| `:digest`      | A content-addressable identifier                  | `WithDigest`                                           |
| `:session`     | A session ID for uploads to the repository        | `WithSessionID`                                        |
| `:ref`         | A tag or digest                                   | `WithRef`                                              |


### Simple example

The following is a simple program that will make several requests to various paths at Quay.io

```go
// Usage: go run example.go <cloud> <bucket> <file>

package main

import (
	"fmt"

	"github.com/bloodorangeio/reggie"
)

func main() {
    client, err := reggie.NewClient("https://quay.io", reggie.WithUsernamePassword("username", "password"), 
        reggie.WithDefaultName("my/repo"))
    if err != nil {
        panic(err)
    }

    req := client.NewRequest(reggie.GET, "/v2/:name/tags/list")
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Response body:\n%v", resp)
}

```

