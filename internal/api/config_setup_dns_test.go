package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestSetupDNSUnsafeProxyReturns400WithoutWriting(t *testing.T) {
	const original = `{"outbounds":[{"type":"direct","tag":"direct"},{"type":"selector","tag":"proxy","outbounds":["direct"]}],"route":{"final":"proxy"}}`
	for _, action := range []string{"preview", "apply"} {
		t.Run(action, func(t *testing.T) {
			handler, path := setupHandler(t, original)
			kernel := &setupKernel{running: true}
			handler.instance = kernel
			body, err := json.Marshal(core.SetupRequest{Modules: []string{"dns"}, SourceHash: core.ConfigContentHash([]byte(original))})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			request := jsonRequest(http.MethodPost, "/", string(body))
			if action == "preview" {
				handler.PreviewSetup(response, request)
			} else {
				handler.ApplySetup(response, request)
			}
			var envelope model.APIResponse
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusBadRequest || envelope.Error == nil || envelope.Error.Code != "config_invalid" {
				t.Fatalf("expected configuration error 400: %d %s", response.Code, response.Body.String())
			}
			if !strings.Contains(envelope.Error.Message, "remove direct fallback") || kernel.reloads != 0 {
				t.Fatalf("rejected DNS setup lost its hint or reloaded: %+v", envelope)
			}
			saved, err := os.ReadFile(path)
			if err != nil || !bytes.Equal([]byte(original), saved) {
				t.Fatalf("rejected DNS setup rewrote user proxy: %s, %v", saved, err)
			}
		})
	}
}
