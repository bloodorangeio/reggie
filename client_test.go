package reggie

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	authUseAccessToken         bool
	lastCapturedRequest        *http.Request
	lastCapturedRequestBodyStr string
)

func TestClient(t *testing.T) {
	authTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		h := r.Header.Get("Authorization")
		if h != expectedAuthHeader {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
			if authUseAccessToken {
				w.Write([]byte(`{"access_token": "abc123"}`))
			} else {
				w.Write([]byte(`{"token": "abc123"}`))
			}
		}
	}))
	defer authTestServer.Close()

	registryTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastCapturedRequest = r
		buf := new(bytes.Buffer)
		buf.ReadFrom(lastCapturedRequest.Body)
		lastCapturedRequestBodyStr = buf.String()

		h := r.Header.Get("Authorization")
		if h == "Bearer abc123" {
			w.Header().Set("Location", "http://abc123location.io/v2/blobs/uploads/e361aeb8-3181-11ea-850d-2e728ce88125")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
    {
        "errors": [{
                "code": "BLOB_UNKNOWN",
                "message": "blob unknown to registry",
                "detail": "lol"
            }
        ]
    }`))
		} else {
			wwwHeader := fmt.Sprintf("Bearer realm=\"%s/v2/auth\",service=\"testservice\",scope=\"testscope\"",
				authTestServer.URL)
			w.Header().Set("www-authenticate", wwwHeader)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer registryTestServer.Close()

	// test client without Address is invalid
	client, err := NewClient("")
	if err == nil {
		t.Fatalf("Providing empty Address should throw error: %s", err)
	}

	// test creating generic client
	client, err = NewClient(registryTestServer.URL,
		WithUsernamePassword("testuser", "testpass"),
		WithDefaultName("testname"),
		WithUserAgent("reggie-tests"))
	if err != nil {
		t.Fatalf("Errors creating client: %s", err)
	}

	// test setting debug option
	client2, err := NewClient(registryTestServer.URL, WithDebug(true))
	if err != nil {
		t.Fatalf("Errors creating client: %s", err)
	}

	if !client2.Config.Debug {
		t.Errorf("Setting the debug flag didn't work")
	}

	// test setting auth scope
	testScope := `realm="https://silly.com/v2/auth",service="testservice",scope="pull,push"`
	client3, err := NewClient(registryTestServer.URL, WithAuthScope(testScope))
	if err != nil {
		t.Fatalf("Errors creating client: %s", err)
	}

	if s := client3.Config.AuthScope; s != testScope {
		t.Errorf("Setting the auth scope didn't work: %s", s)
	}

	// test default name
	req := client.NewRequest(GET, "/v2/<name>/tags/list")
	if !strings.HasSuffix(req.URL, "/v2/testname/tags/list") {
		t.Fatalf("NewRequest does not add default namespace to URL")
	}

	// check user agent
	uaHeader := req.Header.Get("User-Agent")
	if uaHeader != "reggie-tests" {
		t.Fatalf("Expected User-Agent header to be \"reggie-tests\" but instead got \"%s\"", uaHeader)
	}

	resp, responseErr := client.Do(req)
	if responseErr != nil {
		t.Fatalf("Errors executing request: %s", err)
	}
	if status := resp.StatusCode(); status != http.StatusOK {
		t.Fatalf("Expected response code 200 but was %d", status)
	}

	// test default name reset
	client.SetDefaultName("othername")
	req = client.NewRequest(GET, "/v2/<name>/tags/list")
	if !strings.HasSuffix(req.URL, "/v2/othername/tags/list") {
		t.Fatalf("NewRequest does not add runtime namespace to URL")
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Errors executing request: %s", err)
	}
	if status := resp.StatusCode(); status != http.StatusOK {
		t.Fatalf("Expected response code 200 but was %d", status)
	}

	// test custom name on request
	req = client.NewRequest(GET, "/v2/<name>/tags/list", WithName("customname"))
	if !strings.HasSuffix(req.URL, "/v2/customname/tags/list") {
		t.Fatalf("NewRequest does not add runtime namespace to URL")
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Errors executing request: %s", err)
	}
	if status := resp.StatusCode(); status != http.StatusOK {
		t.Fatalf("Expected response code 200 but was %d", status)
	}

	// test Location header on request
	relativeLocation := resp.GetRelativeLocation()
	if strings.Contains(relativeLocation, "http://") || strings.Contains(relativeLocation, "https://") {
		t.Fatalf("Relative Location contains host")
	}
	if relativeLocation == "" {
		t.Fatalf("Location header not present")
	}

	// test error function on response
	errorList, err := resp.Errors()
	if err != nil {
		t.Fatalf("Errors parsing json: %s", err)
	}
	if len(errorList) == 0 {
		t.Fatalf("Error list is length zero")
	}
	e1 := errorList[0]
	if e1.Code == "" {
		t.Fatalf("Code not returned in response body")
	}
	if e1.Message == "" {
		t.Fatalf("Message not returned in response body")
	}
	if e1.Detail == "" {
		t.Fatalf("Detail not returned in response body")
	}

	// test absolute location as well
	absoluteLocation := resp.GetAbsoluteLocation()
	if absoluteLocation == "" {
		t.Fatalf("Location header not present")
	}

	// test reference on request
	req = client.NewRequest(HEAD, "/v2/<name>/manifests/<reference>", WithReference("silly"))
	if !strings.HasSuffix(req.URL, "/v2/othername/manifests/silly") {
		t.Fatalf("NewRequest does not add runtime reference to URL")
	}

	// test digest on request
	digest := "6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401"
	req = client.NewRequest(GET, "/v2/<name>/blobs/<digest>", WithDigest(digest))
	if !strings.HasSuffix(req.URL, fmt.Sprintf("/v2/othername/blobs/%s", digest)) {
		t.Fatalf("NewRequest does not add runtime digest to URL")
	}

	// test session id on request
	id := "f0ca5d12-5557-4747-9c21-3d916f2fc885"
	req = client.NewRequest(GET, "/v2/<name>/blobs/uploads/<session_id>", WithSessionID(id))
	if !strings.HasSuffix(req.URL, fmt.Sprintf("/v2/othername/blobs/uploads/%s", id)) {
		t.Fatalf("NewRequest does not add runtime digest to URL")
	}

	// invalid request (no ref)
	req = client.NewRequest(HEAD, "/v2/<name>/manifests/<reference>")
	resp, err = client.Do(req)
	if err == nil {
		t.Fatalf("Expected error with missing ref")
	}

	// invalid request (no digest)
	req = client.NewRequest(GET, "/v2/<name>/blobs/<digest>")
	resp, err = client.Do(req)
	if err == nil {
		t.Fatalf("Expected error with missing digest")
	}

	// invalid request (no session id)
	req = client.NewRequest(GET, "/v2/<name>/blobs/uploads/<session_id>")
	resp, err = client.Do(req)
	if err == nil {
		t.Fatalf("Expected error with missing session id")
	}

	// bad address on client should not create
	_, err = NewClient("xwejknxw://jshnws")
	if err == nil {
		t.Fatalf("Should be errors creating client with bad address: %s", err)
	}

	// Make sure headers and body match after going through auth
	req = client.NewRequest(PUT, "/a/b/c").
		SetHeader("Content-Length", "3").
		SetHeader("Content-Range", "0-2").
		SetHeader("Content-Type", "application/octet-stream").
		SetQueryParam("digest", "xyz").
		SetBody([]byte("abc"))
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Errors executing request: %s", err)
	}

	// 5 headers expected: the ones we set plus "Authorization" and "User-Agent"
	numHeaders := len(lastCapturedRequest.Header)
	if numHeaders != 5 {
		fmt.Println(lastCapturedRequest.Header)
		t.Fatalf("Expected 5 headers total, instead got %d", numHeaders)
	}

	// Just to be safe, lets check each of the 5 headers are ones we expect
	for _, h := range []string{
		"Content-Length",
		"Content-Range",
		"Content-Type",
		"Authorization",
		"User-Agent",
	} {
		if lastCapturedRequest.Header.Get(h) == "" {
			t.Fatalf("Missing header: %s", h)
		}
	}

	// Check that the body did not get lost somewhere
	if lastCapturedRequestBodyStr != "abc" {
		t.Fatalf("Expected body to be \"abc\" but instead got %s", lastCapturedRequestBodyStr)
	}

	// Test that the retry callback is invoked, if configured.
	newBody := "not the original body"
	req = client.NewRequest(PUT, "/a/b/c", WithRetryCallback(func(r *Request) error {
		r.SetBody(newBody)
		return nil
	})).SetBody([]byte("original body"))
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Errors executing request: %s", err)
	}

	// Ensure the request ended up with the new body.
	if lastCapturedRequestBodyStr != newBody {
		t.Fatalf("Expected body to be %q, but instead got %q", newBody, lastCapturedRequestBodyStr)
	}

	// Test the case where the retry callback returns an error.
	req = client.NewRequest(PUT, "/a/b/c", WithRetryCallback(func(r *Request) error {
		return errors.New("uh oh")
	})).SetBody([]byte("original body"))
	_, err = client.Do(req)
	if err == nil {
		t.Fatalf("Expected error from callback function, but request returned no error")
	}
	if !strings.Contains(err.Error(), "uh oh") {
		t.Fatalf("Expected error to contain callback error \"uh oh\", but instead got %q", err)
	}

	// test for access_token vs. token
	authUseAccessToken = true
	req = client.NewRequest(GET, "/v2/<name>/tags/list")
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("Errors executing request: %s", err)
	}
}
