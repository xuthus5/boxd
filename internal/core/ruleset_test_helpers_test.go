package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

func rulesetTestURL(server *httptest.Server, path string) string {
	_ = server
	return "https://ruleset.test" + path
}

func rulesetTestClient(server *httptest.Server) *http.Client {
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		forwarded := req.Clone(req.Context())
		forwarded.URL.Scheme = serverURL.Scheme
		forwarded.URL.Host = serverURL.Host
		forwarded.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(forwarded)
	})}
}
