package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

var invalidURLTestDefaultsRequests = []struct {
	name string
	body string
}{
	{name: "invalid json", body: `{`},
	{name: "missing enabled", body: `{"url":"https://example.com/test","interval":"3m","tolerance":50}`},
	{name: "null enabled", body: `{"enabled":null,"url":"https://example.com/test","interval":"3m","tolerance":50}`},
	{name: "missing url", body: `{"enabled":true,"interval":"3m","tolerance":50}`},
	{name: "null url", body: `{"enabled":true,"url":null,"interval":"3m","tolerance":50}`},
	{name: "missing interval", body: `{"enabled":true,"url":"https://example.com/test","tolerance":50}`},
	{name: "null interval", body: `{"enabled":true,"url":"https://example.com/test","interval":null,"tolerance":50}`},
	{name: "missing tolerance", body: `{"enabled":true,"url":"https://example.com/test","interval":"3m"}`},
	{name: "null tolerance", body: `{"enabled":true,"url":"https://example.com/test","interval":"3m","tolerance":null}`},
	{name: "invalid url", body: `{"enabled":true,"url":"ftp://example.com/test","interval":"3m","tolerance":50}`},
	{name: "invalid interval", body: `{"enabled":true,"url":"https://example.com/test","interval":"0s","tolerance":50}`},
	{name: "tolerance overflow", body: `{"enabled":true,"url":"https://example.com/test","interval":"3m","tolerance":65536}`},
}

func TestSettingsHandlerURLTestDefaults(t *testing.T) {
	_, _, settings, _ := newAPIManagers(t)
	handler := NewSettingsHandler(settings)

	rr := httptest.NewRecorder()
	handler.GetURLTestDefaults(rr, httptest.NewRequest(http.MethodGet, "/api/settings/urltest-defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("default get status = %d, want %d", rr.Code, http.StatusOK)
	}
	defaults := decodeBody[model.URLTestDefaults](t, rr)
	if !defaults.Enabled || defaults.Interval != "3m" || defaults.Tolerance != 50 {
		t.Fatalf("default response = %#v", defaults)
	}

	rr = httptest.NewRecorder()
	handler.SetURLTestDefaults(rr, jsonRequest(
		http.MethodPut,
		"/api/settings/urltest-defaults",
		`{"enabled":false,"url":"https://example.com/generate_204","interval":"5m","tolerance":0}`,
	))
	if rr.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", rr.Code, rr.Body.String())
	}
	updated := decodeBody[model.URLTestDefaults](t, rr)
	if updated.Enabled || updated.URL != "https://example.com/generate_204" || updated.Interval != "5m" || updated.Tolerance != 0 {
		t.Fatalf("put response = %#v", updated)
	}

	rr = httptest.NewRecorder()
	handler.GetURLTestDefaults(rr, httptest.NewRequest(http.MethodGet, "/api/settings/urltest-defaults", nil))
	stored := decodeBody[model.URLTestDefaults](t, rr)
	if stored != updated {
		t.Fatalf("stored response = %#v, want %#v", stored, updated)
	}
}

func TestSettingsHandlerSetURLTestDefaultsErrors(t *testing.T) {
	_, _, settings, _ := newAPIManagers(t)
	handler := NewSettingsHandler(settings)

	for _, tt := range invalidURLTestDefaultsRequests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.SetURLTestDefaults(rr, jsonRequest(
				http.MethodPut,
				"/api/settings/urltest-defaults",
				tt.body,
			))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
			}
		})
	}
}
