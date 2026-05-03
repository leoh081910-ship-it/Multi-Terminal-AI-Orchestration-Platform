package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWSHub(t *testing.T) *WSHub {
	t.Helper()
	hub := NewWSHub(zerolog.Nop())
	go hub.Run()
	t.Cleanup(func() {
		// Hub runs forever; GC will clean up after test
	})
	return hub
}

func TestWSHubConnectAndBroadcast(t *testing.T) {
	hub := newTestWSHub(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())

	// Broadcast a message
	hub.Broadcast(WSMessage{
		Type:      "test.event",
		Data:      map[string]string{"hello": "world"},
		Timestamp: time.Now().UTC(),
	})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var received WSMessage
	require.NoError(t, json.Unmarshal(msg, &received))
	assert.Equal(t, "test.event", received.Type)
}

func TestWSHubAuthRequired(t *testing.T) {
	hub := newTestWSHub(t)
	hub.SetAuthFunc(func(token string) (string, bool) {
		if token == "valid-token" {
			return "user-1", true
		}
		return "", false
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// No token -> 401
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Valid token via query param -> success
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	require.NoError(t, err)
	conn.Close()

	// Invalid token -> 401
	resp2, err := http.Get(srv.URL + "?token=bad-token")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)
}

func TestWSHubProjectFilter(t *testing.T) {
	hub := newTestWSHub(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Client subscribed to project A
	connA, _, err := websocket.DefaultDialer.Dial(wsURL+"?project=proj-a", nil)
	require.NoError(t, err)
	defer connA.Close()

	// Client subscribed to all
	connAll, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer connAll.Close()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, hub.ClientCount())

	// Broadcast to project A
	hub.Broadcast(WSMessage{
		Type:      "task.update",
		ProjectID: "proj-a",
		Timestamp: time.Now().UTC(),
	})

	// Client A should receive
	connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := connA.ReadMessage()
	require.NoError(t, err)
	var wsMsg WSMessage
	require.NoError(t, json.Unmarshal(msg, &wsMsg))
	assert.Equal(t, "task.update", wsMsg.Type)

	// Client all should also receive
	connAll.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg2, err := connAll.ReadMessage()
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(msg2, &wsMsg))
	assert.Equal(t, "task.update", wsMsg.Type)
}

func TestExtractWSToken(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(r *http.Request)
		expect string
	}{
		{"query param", func(r *http.Request) { r.URL.RawQuery = "token=abc123" }, "abc123"},
		{"auth header", func(r *http.Request) { r.Header.Set("Authorization", "Bearer xyz789") }, "xyz789"},
		{"empty", func(r *http.Request) {}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			tt.setup(req)
			assert.Equal(t, tt.expect, extractWSToken(req))
		})
	}
}
