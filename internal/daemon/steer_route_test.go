package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/engine/streaming"
	"github.com/GrayCodeAI/reflect/internal/testutil"
)

func postSteer(t *testing.T, addr, id, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+addr+"/v1/sessions/"+id+"/steer", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new steer request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /v1/sessions/%s/steer: %v", id, err)
	}
	return resp
}

// TestDaemon_SteerEndpoint verifies POST /v1/sessions/{id}/steer enqueues
// guidance on the live generation's steering queue and returns 404 once the
// generation ends. Requests stay within the per-IP API burst (4).
func TestDaemon_SteerEndpoint(t *testing.T) {
	srv := New(Config{Port: 0, Host: testutil.LoopbackHost}, daemonTestSessionFactory(nil))
	addr := startTestDaemon(t, srv)
	defer srv.Stop(context.Background())

	sq := streaming.NewSteeringQueue()
	srv.registerSteerer("steer-sess-1", sq)

	resp := postSteer(t, addr, "steer-sess-1", `{"message":"focus on the auth tests"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from steer, got %d", resp.StatusCode)
	}
	var body map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		resp.Body.Close()
		t.Fatalf("decode steer response: %v", err)
	}
	resp.Body.Close()
	if !body["steered"] {
		t.Error("expected steered=true")
	}
	drained := sq.Drain()
	if len(drained) != 1 || drained[0].Content != "focus on the auth tests" {
		t.Fatalf("steering queue after POST = %+v, want one guidance message", drained)
	}

	// Invalid session id is rejected before any queue lookup.
	resp = postSteer(t, addr, "bad!id", `{"message":"x"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid session id, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Unknown session has no active generation.
	resp = postSteer(t, addr, "never-seen", `{"message":"x"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown session, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Once the generation ends (entry unregistered), steering 404s.
	srv.unregisterSteerer("steer-sess-1", sq)
	resp = postSteer(t, addr, "steer-sess-1", `{"message":"too late"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after generation ended, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestDaemon_SteerEndpointRejectsBadBodies covers body validation while a
// generation is live, on its own server to stay within the per-IP burst.
func TestDaemon_SteerEndpointRejectsBadBodies(t *testing.T) {
	srv := New(Config{Port: 0, Host: testutil.LoopbackHost}, daemonTestSessionFactory(nil))
	addr := startTestDaemon(t, srv)
	defer srv.Stop(context.Background())

	sq := streaming.NewSteeringQueue()
	srv.registerSteerer("steer-sess-1", sq)

	for name, payload := range map[string]string{
		"empty message":  `{"message":""}`,
		"blank message":  `{"message":"   "}`,
		"malformed json": `{"message":`,
		"unknown field":  `{"message":"x","extra":1}`,
	} {
		resp := postSteer(t, addr, "steer-sess-1", payload)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, resp.StatusCode)
		}
		resp.Body.Close()
	}
	if sq.HasPending() {
		t.Fatal("rejected requests must not enqueue steering")
	}
}

// TestSteerRequiresAuthentication mirrors the graph route's auth test: no or
// wrong bearer token must be rejected before any steering side effect.
func TestSteerRequiresAuthentication(t *testing.T) {
	srv := New(Config{APIKey: "secret"}, nil)
	sq := streaming.NewSteeringQueue()
	srv.registerSteerer("steer-sess-1", sq)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/sessions/steer-sess-1/steer", bytes.NewBufferString(`{"message":"x"}`))
	srv.relay.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if sq.HasPending() {
		t.Fatal("unauthorized request must not enqueue steering")
	}
}
