# Reggie

[![GitHub Actions status](https://github.com/bloodorangeio/reggie/workflows/build/badge.svg)](https://github.com/bloodorangeio/reggie/actions?query=workflow%3Abuild) [![Go Report Card](https://goreportcard.com/badge/github.com/bloodorangeio/reggie)](https://goreportcard.com/report/github.com/bloodorangeio/reggie) [![GoDoc](https://godoc.org/github.com/bloodorangeio/reggie?status.svg)](https://godoc.org/github.com/bloodorangeio/reggie)

![](https://raw.githubusercontent.com/bloodorangeio/reggie/master/reggie.png)

Reggie is a dead simple Go HTTP client designed to be used against [OCI Distribution](https://github.com/opencontainers/distribution-spec), built on top of the following libraries:

- [go-resty/resty](https://github.com/go-resty/resty) - for user-friendly HTTP helper methods
- [genuinetools/reg](https://github.com/genuinetools/reg) - for "Docker-style" auth support

*Note: Authentication/authorization is not part of the distribution spec, but it has been implemented similarly across registry providers targeting the Docker client.*


## Getting Started

First import the library:
```go
import "github.com/bloodorangeio/reggie"
```

Then construct a client:

```go
client, err := reggie.NewClient("http://localhost:5000")
```

You may also construct the client with a number of options related to authentication, etc:

```go
client, err := reggie.NewClient("https://r.mysite.io",
    reggie.WithUsernamePassword("myuser", "mypass"),  // registry credentials
    reggie.WIthDefaultName("myorg/myrepo"),           // default repo name
    reggie.WithDebug())                               // enable debug logging
```

## Making Requests

Reggie uses a domain-specific language to supply various parts of the URI path in order to provide visual parity with [the spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md).

For example, to list all tags for the repo `megacorp/superapp`, you might do the following:

```go
req := client.NewRequest(reggie.GET, "/v2/<name>/tags/list",
    reggie.WithName("megacorp/superapp"))
```

This will result in a request object built for `GET /v2/megacorp/superapp/tags/list`.

You may then use any of the methods provided by [resty](https://github.com/go-resty/resty) to modify the request:
```go
req.SetQueryParam("n", "20")  // example: tag pagination, first 20 results
```

Finally, execute the request, which will return a resty-based response object:
```go
resp, err := client.Do(req)
fmt.Println("Status Code:", resp.StatusCode())
```

## Path Substitutions

Below is a table of all of the possible URI parameter substitutions and associated methods:


| URI Parameter | Description | Option method |
|-|-|-|
| `<name>` | Namespace of a repository within the registry | `WithDefaultName` (`Client`) or `WithName` (`Request`) |
| `<digest>` | Content-addressable identifier | `WithDigest` (`Request`) |
| `<reference>` | Tag or digest | `WithReference` (`Request`) |
| `<session_id>` | Session ID for upload | `WithSessionID` (`Request`) |

## Example

The following is an example of a resumable blob upload and subsequent manifest upload:

```
// TODO
```

## Other Features

### Error Parsing

On the response object, you may call the `Errors()` method which will attempt to parse the response body into a list of [OCI ErrorInfo](https://github.com/opencontainers/distribution-spec/blob/master/specs-go/v1/error.go#L36) objects:
```go
for _, e := range resp.Errors() {
    fmt.Println("Code:",    e.Code)
    fmt.Println("Message:", e.Message)
    fmt.Println("Detail:",  e.Detail)
}
```

### Location Header Parsing

For certain types of requests, such as chunked uploads, the `Location` header is needed in order to make follow-up requests.

Reggie provides two helper methods to obtain the redirect location:
```go
fmt.Println("Relative location:", resp.RelativeLocation())  // /v2/...
fmt.Println("Absolute location:", resp.AbsoluteLocation())  // https://...
```

### HTTP Method Constants

Simply-named constants are provided for the following HTTP request methods:
```go
reggie.GET     // "GET"
reggie.PUT     // "PUT"
reggie.PATCH   // "PATCH"
reggie.DELETE  // "DELETE"
reggie.POST    // "POST"
reggie.HEAD    // "HEAD"
reggie.OPTIONS // "OPTIONS"
```

### Custom User-Agent

Requests made by Reggie will use a custom value by default for the `User-Agent` header in order for registry providers to identify incoming requests:
```
User-Agent: reggie/0.1.1 (https://github.com/bloodorangeio/reggie)
```
