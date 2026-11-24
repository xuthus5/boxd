package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/adapters/inbound/httpapi"
	"github.com/xuthus5/boxd/internal/bootstrap"
	"github.com/xuthus5/boxd/internal/kernel/fake"
)

const testPassword = "correct-horse-battery"

type testEnvironment struct {
	app    *bootstrap.App
	server *httptest.Server
	client *http.Client
}

func newTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()
	app, err := bootstrap.New(context.Background(), bootstrap.Options{
		DataDir: t.TempDir(),
		Factory: fake.NewFactory(),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("bootstrap.New() error = %v", err)
	}
	adapter, err := httpapi.New(app.Control, httpapi.Options{
		Address: "127.0.0.1:0",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		_ = app.Close()
		t.Fatalf("httpapi.New() error = %v", err)
	}
	server := httptest.NewServer(adapter.Handler())
	jar, err := cookiejar.New(nil)
	if err != nil {
		server.Close()
		_ = app.Close()
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	environment := &testEnvironment{
		app:    app,
		server: server,
		client: &http.Client{Jar: jar, Timeout: 10 * time.Second},
	}
	t.Cleanup(func() {
		server.Close()
		if err := app.Close(); err != nil {
			t.Errorf("app.Close() error = %v", err)
		}
	})
	return environment
}

func (e *testEnvironment) request(t *testing.T, method, path string, body any, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, e.server.URL+path, reader)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := e.client.Do(request)
	if err != nil {
		t.Fatalf("%s %s error = %v", method, path, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response error = %v", err)
	}
	return response.StatusCode, response.Header.Clone(), payload
}

func decodeObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", body, err)
	}
	return value
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	value := decodeObject(t, body)
	detail, _ := value["error"].(map[string]any)
	code, _ := detail["code"].(string)
	return code
}

func (e *testEnvironment) initialize(t *testing.T) string {
	t.Helper()
	status, _, body := e.request(t, http.MethodPost, "/api/v1/auth/initialize", map[string]string{"password": testPassword}, nil)
	if status != http.StatusCreated {
		t.Fatalf("initialize status = %d, body = %s", status, body)
	}
	token, _ := decodeObject(t, body)["token"].(string)
	if token == "" {
		t.Fatalf("initialize response did not include a session token: %s", body)
	}
	return token
}

func TestAuthenticationCookieBearerAndOriginPolicy(t *testing.T) {
	environment := newTestEnvironment(t)

	status, _, body := environment.request(t, http.MethodGet, "/api/v1/health", nil, nil)
	if status != http.StatusOK || decodeObject(t, body)["service"] != "boxd-web" {
		t.Fatalf("health = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/snapshot", nil, nil)
	if status != http.StatusUnauthorized || errorCode(t, body) != "kernel_unauthorized" {
		t.Fatalf("unauthenticated snapshot = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/auth/status", nil, nil)
	authStatus := decodeObject(t, body)
	if status != http.StatusOK || authStatus["initialized"] != false || authStatus["authenticated"] != false {
		t.Fatalf("initial auth status = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/auth/initialize", map[string]string{"password": "short"}, nil)
	if status != http.StatusBadRequest || errorCode(t, body) != "kernel_invalid_config" {
		t.Fatalf("short password initialize = %d %s", status, body)
	}

	status, headers, body := environment.request(t, http.MethodPost, "/api/v1/auth/initialize", map[string]string{"password": testPassword}, nil)
	if status != http.StatusCreated {
		t.Fatalf("initialize = %d %s", status, body)
	}
	token, _ := decodeObject(t, body)["token"].(string)
	if token == "" {
		t.Fatalf("initialize response did not include a token: %s", body)
	}
	cookie := headers.Get("Set-Cookie")
	if !strings.Contains(cookie, "boxd_session=") || !strings.Contains(cookie, "HttpOnly") || !strings.Contains(cookie, "SameSite=Strict") {
		t.Fatalf("session cookie = %q", cookie)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/auth/status", nil, nil)
	authStatus = decodeObject(t, body)
	if status != http.StatusOK || authStatus["initialized"] != true || authStatus["authenticated"] != true {
		t.Fatalf("authenticated status = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/auth/initialize", map[string]string{"password": testPassword}, nil)
	if status != http.StatusConflict || errorCode(t, body) != "kernel_conflict" {
		t.Fatalf("second initialize = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/snapshot", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("cookie snapshot = %d %s", status, body)
	}
	snapshot := decodeObject(t, body)
	if snapshot["revision"] == "" {
		t.Fatalf("snapshot revision is empty: %s", body)
	}
	if inbounds, ok := snapshot["inbounds"].([]any); !ok || len(inbounds) != 2 {
		t.Fatalf("default inbounds = %#v", snapshot["inbounds"])
	}
	if ruleSets, ok := snapshot["rulesets"].([]any); !ok || len(ruleSets) != 3 {
		t.Fatalf("default rule sets = %#v", snapshot["rulesets"])
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/auth/logout", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("logout = %d %s", status, body)
	}
	status, _, body = environment.request(t, http.MethodGet, "/api/v1/snapshot", nil, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("snapshot after logout = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"password": "wrong-password"}, nil)
	if status != http.StatusUnauthorized || errorCode(t, body) != "kernel_unauthorized" {
		t.Fatalf("wrong login = %d %s", status, body)
	}
	status, _, body = environment.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"password": testPassword}, nil)
	if status != http.StatusOK {
		t.Fatalf("login = %d %s", status, body)
	}
	bearerToken, _ := decodeObject(t, body)["token"].(string)
	if bearerToken == "" {
		t.Fatalf("login response did not include a token: %s", body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/snapshot", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("cookie login snapshot = %d %s", status, body)
	}

	bearerClient := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest(http.MethodGet, environment.server.URL+"/api/v1/snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+bearerToken)
	response, err := bearerClient.Do(request)
	if err != nil {
		t.Fatalf("bearer snapshot error = %v", err)
	}
	bearerBody, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bearer snapshot = %d %s", response.StatusCode, bearerBody)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/settings", map[string]any{
		"settings": map[string]string{"theme": "light"},
	}, map[string]string{"Origin": "https://attacker.example"})
	if status != http.StatusForbidden || errorCode(t, body) != "request_forbidden" {
		t.Fatalf("cross-origin cookie mutation = %d %s", status, body)
	}
}

func TestNodeSubscriptionRedactionAndStaticConsole(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.initialize(t)

	status, _, body := environment.request(t, http.MethodPost, "/api/v1/nodes/import-links", map[string]any{
		"links": []string{"trojan://secret@127.0.0.2:443?sni=example.com#Trojan"},
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("import links = %d %s", status, body)
	}
	imported := decodeObject(t, body)
	if imported["added"] != float64(1) {
		t.Fatalf("import result = %s", body)
	}
	nodes, _ := imported["nodes"].([]any)
	if len(nodes) != 1 {
		t.Fatalf("imported nodes = %s", body)
	}
	node, _ := nodes[0].(map[string]any)
	if node["protocol"] != "trojan" || node["server"] != "127.0.0.2" {
		t.Fatalf("imported node = %#v", node)
	}
	if _, exposed := node["options"]; exposed {
		t.Fatalf("node view exposed protocol options: %#v", node)
	}
	if bytes.Contains(body, []byte("secret")) || bytes.Contains(body, []byte("password")) {
		t.Fatalf("node import response exposed credentials: %s", body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/nodes", nil, nil)
	if status != http.StatusOK || bytes.Contains(body, []byte("secret")) || bytes.Contains(body, []byte("password")) {
		t.Fatalf("node list exposed credentials = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/subscriptions", map[string]any{
		"id":               "sub-redaction",
		"name":             "Redaction subscription",
		"url":              "https://example.com/sub?token=super-secret-token&name=value",
		"enabled":          true,
		"interval_seconds": 3600,
	}, nil)
	if status != http.StatusCreated {
		t.Fatalf("create subscription = %d %s", status, body)
	}
	if bytes.Contains(body, []byte("super-secret-token")) || !bytes.Contains(body, []byte("***")) {
		t.Fatalf("subscription URL was not redacted: %s", body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/snapshot", nil, nil)
	if status != http.StatusOK || bytes.Contains(body, []byte("super-secret-token")) {
		t.Fatalf("snapshot exposed subscription credentials = %d %s", status, body)
	}

	status, headers, body := environment.request(t, http.MethodGet, "/", nil, nil)
	if status != http.StatusOK || !bytes.Contains(body, []byte("BOXD")) {
		t.Fatalf("console index = %d %s", status, body)
	}
	if contentType := headers.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("console Content-Type = %q", contentType)
	}
	if !strings.Contains(headers.Get("Content-Security-Policy"), "default-src 'self'") || headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("security headers = %#v", headers)
	}
	if headers.Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Fatalf("index Cache-Control = %q", headers.Get("Cache-Control"))
	}
}

func TestNodeEditorAndReachabilityBatch(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.initialize(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			_ = connection.Close()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	link := "trojan://secret@" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "?sni=example.com#Editable"
	status, _, body := environment.request(t, http.MethodPost, "/api/v1/nodes/import-links", map[string]any{"links": []string{link}}, nil)
	if status != http.StatusOK {
		t.Fatalf("import links = %d %s", status, body)
	}
	imported := decodeObject(t, body)
	nodes, _ := imported["nodes"].([]any)
	if len(nodes) != 1 {
		t.Fatalf("imported nodes = %s", body)
	}
	nodeID, _ := nodes[0].(map[string]any)["id"].(string)
	if nodeID == "" {
		t.Fatalf("imported node has no ID: %s", body)
	}

	status, _, body = environment.request(t, http.MethodGet, "/api/v1/nodes/"+nodeID, nil, nil)
	editor := decodeObject(t, body)
	if status != http.StatusOK || editor["name"] != "Editable" || editor["options"] == nil {
		t.Fatalf("node editor = %d %s", status, body)
	}

	status, _, body = environment.request(t, http.MethodPut, "/api/v1/nodes/"+nodeID, map[string]any{
		"name":            "Edited node",
		"protocol":        "trojan",
		"server":          "127.0.0.1",
		"port":            port,
		"subscription_id": "",
		"options":         map[string]any{"password": "secret", "sni": "example.com"},
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("update node = %d %s", status, body)
	}
	if updated := decodeObject(t, body); updated["name"] != "Edited node" {
		t.Fatalf("updated node = %s", body)
	}

	status, _, body = environment.request(t, http.MethodPost, "/api/v1/nodes/test", map[string]any{
		"node_ids": []string{nodeID},
		"kind":     "tcp",
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("node tcp test = %d %s", status, body)
	}
	results, _ := decodeObject(t, body)["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("node tcp results = %s", body)
	}
	result, _ := results[0].(map[string]any)
	if result["node_id"] != nodeID || result["kind"] != "tcp" || result["error"] != nil {
		t.Fatalf("node tcp result = %#v", result)
	}
}

func TestEventStreamDeliversReadyEvent(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.initialize(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, environment.server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := environment.client.Do(request)
	if err != nil {
		t.Fatalf("SSE request error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("SSE Content-Type = %q", contentType)
	}
	reader := bufio.NewReader(response.Body)
	var sawReady bool
	for index := 0; index < 12; index++ {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("read SSE line error = %v", readErr)
		}
		if strings.TrimSpace(line) == "event: ready" {
			sawReady = true
			break
		}
	}
	if !sawReady {
		t.Fatal("SSE stream did not emit the ready event")
	}
}
