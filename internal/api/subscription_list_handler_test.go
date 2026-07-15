package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/core"
)

func TestSubscriptionBackedListHandlersReturnInternalErrorForInvalidData(t *testing.T) {
	nodeManager, subscriptionManager, _, configPath := newAPIManagers(t)
	putInvalidSubscription(t, subscriptionManager)

	tests := []struct {
		name   string
		handle func(http.ResponseWriter, *http.Request)
		path   string
	}{
		{
			name:   "subscriptions",
			handle: NewSubscriptionHandler(subscriptionManager, nodeManager, configPath).List,
			path:   "/api/subscriptions",
		},
		{
			name:   "nodes",
			handle: NewNodesHandler(nodeManager, subscriptionManager, configPath).List,
			path:   "/api/nodes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.handle(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
			}
		})
	}
}

func putInvalidSubscription(t *testing.T, manager *core.SubscriptionManager) {
	t.Helper()
	if err := manager.DB().Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte("broken"), []byte("{"))
	}); err != nil {
		t.Fatalf("saving invalid subscription: %v", err)
	}
}
