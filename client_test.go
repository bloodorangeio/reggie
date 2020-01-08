package reggie

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient(t *testing.T) {
	authTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
		h := r.Header.Get("Authorization")
		if h != expectedAuthHeader {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "abc123"}`))
		}
	}))
	defer authTestServer.Close()

	registryTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	client, err := NewClient(registryTestServer.URL,
		WithUsernamePassword("testuser", "testpass"),
		WithDefaultName("testname"))
	if err != nil {
		t.Fatalf("Errors creating client: %s", err)
	}

	//test setting debug option
	client2, err := NewClient(registryTestServer.URL, WithDebug(true))
	if err != nil {
		t.Fatalf("Errors creating client: %s", err)
	}

	if !client2.Config.Debug {
		t.Errorf("Setting the debug flag didn't work")
	}

	// test default name
	req := client.NewRequest(GET, "/v2/<name>/tags/list")
	if !strings.HasSuffix(req.URL, "/v2/testname/tags/list") {
		t.Fatalf("NewRequest does not add default namespace to URL")
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
	e, err := resp.Errors()
	if err != nil {
		t.Fatalf("Errors parsing json: %s", err)
	}
	if e.Code() == "" {
		t.Fatalf("Code not returned in response body")
	}
	if e.Message() == "" {
		t.Fatalf("Message not returned in response body")
	}
	if e.Detail() == "" {
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

	// bad address on client
	badClient, err := NewClient("xwejknxw://jshnws")
	if err != nil {
		t.Fatalf("Errors creating client with bad address: %s", err)
	}
	req = badClient.NewRequest(GET, "/v2/<name>/tags/list", WithName("customname"))
	resp, err = badClient.Do(req)
	if err == nil {
		t.Fatalf("Expected error with bad address")
	}

}
