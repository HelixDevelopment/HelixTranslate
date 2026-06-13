package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"

	"github.com/gorilla/websocket"
)

// TestServerHandler_SessionFilteringHonored is the §11.4.115 reproduce-first /
// regression guard for the StartServer fan-out bug.
//
// Root cause (FACT, pre-fix): Hub.StartServer's handler built the Client
// reading only client_id from the query and left Client.SessionID == "".
// handleEvent skips the per-session filter when client.SessionID == "", so a
// dashboard that connected for session "alpha" received EVERY session's events
// (including "beta"). The canonical cmd/monitor-server set SessionID correctly;
// the hub's own server method did not, and cmd/ssh-translation uses it.
//
// RED on the broken handler: the client receives the "beta" event -> FAIL.
// GREEN after the fix (wsHandler reads session_id): only "alpha" arrives.
func TestServerHandler_SessionFilteringHonored(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	srv := httptest.NewServer(http.HandlerFunc(hub.wsHandler))
	defer srv.Close()

	// Connect as session "alpha" via the real query-string path the dashboard uses.
	wsURL := "ws" + srv.URL[4:] + "/ws?session_id=alpha&client_id=dash1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Wait for async registration to settle.
	deadline := time.Now().Add(2 * time.Second)
	for hub.GetClientCount() != 1 {
		if time.Now().After(deadline) {
			t.Fatal("client never registered")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Publish a foreign-session event FIRST. A correctly-filtered "alpha" client
	// must NOT receive this.
	betaEvent := events.NewEvent(events.EventTranslationProgress, "BETA-LEAK", nil)
	betaEvent.SessionID = "beta"
	eventBus.Publish(betaEvent)

	// Then publish the client's own-session event.
	alphaEvent := events.NewEvent(events.EventTranslationProgress, "ALPHA-OK", nil)
	alphaEvent.SessionID = "alpha"
	eventBus.Publish(alphaEvent)

	// Read exactly one frame. With the bug it is "beta" (FAIL); with the fix the
	// beta event is filtered out and the first frame is "alpha".
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got events.Event
	if err := json.Unmarshal(msg, &got); err != nil {
		t.Fatalf("unmarshal %q: %v", string(msg), err)
	}

	if got.SessionID != "alpha" {
		t.Fatalf("session-filtering hole: alpha client received session %q event (message=%q); "+
			"StartServer handler must set Client.SessionID from the session_id query param",
			got.SessionID, got.Message)
	}
}
