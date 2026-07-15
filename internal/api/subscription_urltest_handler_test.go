package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

func TestSubscriptionHandlerCreatesURLTestOverrides(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath)
	created := createSubscriptionWithURLTest(t, handler)
	if created.URLTest == nil || created.URLTest.Enabled == nil || *created.URLTest.Enabled {
		t.Fatalf("created urltest overrides = %#v", created.URLTest)
	}
	if created.URLTest.Tolerance == nil || *created.URLTest.Tolerance != 0 {
		t.Fatalf("created tolerance = %#v", created.URLTest.Tolerance)
	}
}

func TestSubscriptionHandlerClearsURLTestOverrides(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath)
	created := createSubscriptionWithURLTest(t, handler)
	recorder := httptest.NewRecorder()
	request := withURLParam(jsonRequest(
		http.MethodPut,
		"/api/subscriptions/"+created.ID,
		`{"name":"custom","url":"https://example.com/sub","interval_min":60,"urltest":null}`,
	), "id", created.ID)
	handler.Update(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("clear status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if stored := subscriptionManager.Get(created.ID); stored == nil || stored.URLTest != nil {
		t.Fatalf("stored subscription = %#v", stored)
	}
}

func TestSubscriptionHandlerRejectsInvalidURLTestOverrides(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	handler := NewSubscriptionHandler(subscriptionManager, nodeManager, configPath)
	recorder := httptest.NewRecorder()
	handler.Create(recorder, jsonRequest(
		http.MethodPost,
		"/api/subscriptions",
		`{"name":"invalid","url":"https://example.com/sub","interval_min":60,"urltest":{"url":"file:///tmp/test"}}`,
	))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func createSubscriptionWithURLTest(t *testing.T, handler *SubscriptionHandler) model.Subscription {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.Create(recorder, jsonRequest(
		http.MethodPost,
		"/api/subscriptions",
		`{"name":"custom","url":"https://example.com/sub","interval_min":60,"urltest":{"enabled":false,"tolerance":0}}`,
	))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	return decodeBody[model.Subscription](t, recorder)
}
