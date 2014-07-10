package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// CDNBackendServer instance should be ready to serve requests when test
// suite starts and then serve custom handlers each with their own status
// code.
func TestHelpersCDNBackendServerHandlers(t *testing.T) {
	ResetBackends(backendsByPriority)

	url := originServer.server.URL + "/" + NewUUID()
	req, _ := http.NewRequest("GET", url, nil)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != 200 {
		t.Error("First request to default handler failed")
	}

	for _, statusCode := range []int{301, 302, 403, 404} {
		originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		})

		resp := RoundTripCheckError(t, req)
		if resp.StatusCode != statusCode {
			t.Errorf("SwitchHandler didn't work. Got %d, expected %d", resp.StatusCode, statusCode)
		}
	}
}

// CDNBackendServer should always respond to HEAD requests in order for the
// CDN to determine the health of our origin.
func TestHelpersCDNBackendServerProbes(t *testing.T) {
	ResetBackends(backendsByPriority)

	originServer.SwitchHandler(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HEAD request incorrectly served by CDNBackendServer.handler")
	})

	url := originServer.server.URL + "/"
	req, _ := http.NewRequest("HEAD", url, nil)
	resp := RoundTripCheckError(t, req)

	if resp.StatusCode != 200 || resp.Header.Get("PING") != "PONG" {
		t.Error("HEAD request for '/' served incorrectly")
	}
}

func TestHelpersCDNServeStop(t *testing.T) {
	ResetBackends(backendsByPriority)

	var expectedStarted bool

	expectedStarted = true
	if started := originServer.IsStarted(); started != expectedStarted {
		t.Errorf(
			"originServer.IsStarted() incorrect. Expected %t, got %t",
			expectedStarted,
			started,
		)
	}

	url := originServer.server.URL + "/" + NewUUID()
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error("originServer should be up and responding, prior to Stop operation")
	}

	originServer.Stop()
	expectedStarted = false
	if started := originServer.IsStarted(); started {
		t.Errorf(
			"originServer.IsStarted() incorrect. Expected %t, got %t",
			expectedStarted,
			started,
		)
	}

	resp, err = client.RoundTrip(req)
	if err == nil {
		t.Error("Client connection succeeded. The server should be refusing requests by now.")
	}

	re := regexp.MustCompile(`EOF`)
	if !re.MatchString(fmt.Sprintf("%s", err)) {
		t.Errorf("Connection error %q is not as expected", err)
	}
}

// CDNBackendServer should use TLS by default as evidenced by an HTTPS URL
// from `httptest.Server`.
func TestHelpersCDNBackendServerTLSEnabled(t *testing.T) {
	const disableTLS = false
	const expectedURLPrefix = "https://"

	backend := CDNBackendServer{
		Name: "test",
		Port: 0,
	}

	if backend.TLSDisabled != disableTLS {
		t.Errorf(
			"TLSDisabled should not be enabled by default. Expected %t, got %t",
			disableTLS,
			backend.TLSDisabled,
		)
	}

	backend.Start()
	defer backend.Stop()

	if url := backend.server.URL; !strings.HasPrefix(url, expectedURLPrefix) {
		t.Errorf(
			"Expected backend URL to begin with %q, got %q",
			expectedURLPrefix,
			url,
		)
	}
}

// CDNBackendServer should disable TLS if passed `TLSDisabled: true`, as
// evidenced by an HTTP URL from `httptest.Server`.
func TestHelpersCDNBackendServerTLSDisabled(t *testing.T) {
	const disableTLS = true
	const expectedURLPrefix = "http://"

	backend := CDNBackendServer{
		Name:        "test",
		Port:        0,
		TLSDisabled: disableTLS,
	}

	backend.Start()
	defer backend.Stop()

	if url := backend.server.URL; !strings.HasPrefix(url, expectedURLPrefix) {
		t.Errorf(
			"Expected backend URL to begin with %q, got %q",
			expectedURLPrefix,
			url,
		)
	}
}
