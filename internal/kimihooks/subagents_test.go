package kimihooks

import (
	"path/filepath"
	"testing"

	"github.com/hellolib/agent-notify/internal/notify"
)

func TestSubagentStoreTracksNestedStarts(t *testing.T) {
	store := newSubagentStore(filepath.Join(t.TempDir(), "kimi-subagents.json"))
	const sid = "sess-1"

	if store.Active(sid) {
		t.Fatal("empty store should be inactive")
	}
	store.Begin(sid)
	store.Begin(sid)
	if !store.Active(sid) {
		t.Fatal("want active after two begins")
	}
	store.End(sid)
	if !store.Active(sid) {
		t.Fatal("want still active after one end")
	}
	store.End(sid)
	if store.Active(sid) {
		t.Fatal("want inactive after matching ends")
	}
}

func TestTrackSubagentLifecycleDropsStopWhileActive(t *testing.T) {
	store := newSubagentStore(filepath.Join(t.TempDir(), "kimi-subagents.json"))
	const sid = "sess-2"

	drop, _ := trackSubagentLifecycle(store, notify.Message{Event: "subagent_start", SessionID: sid})
	if !drop {
		t.Fatal("subagent_start should drop")
	}
	drop, reason := trackSubagentLifecycle(store, notify.Message{Event: "run_completed", SessionID: sid})
	if !drop {
		t.Fatal("run_completed should drop while subagent is running")
	}
	if reason == "" {
		t.Fatal("want skip reason")
	}
	drop, _ = trackSubagentLifecycle(store, notify.Message{Event: "subagent_stop", SessionID: sid})
	if !drop {
		t.Fatal("subagent_stop should drop")
	}
	drop, _ = trackSubagentLifecycle(store, notify.Message{Event: "run_completed", SessionID: sid})
	if drop {
		t.Fatal("main-agent run_completed should not drop")
	}
}

func TestTrackSubagentLifecycleSessionStartResets(t *testing.T) {
	store := newSubagentStore(filepath.Join(t.TempDir(), "kimi-subagents.json"))
	const sid = "sess-3"
	store.Begin(sid)
	drop, _ := trackSubagentLifecycle(store, notify.Message{Event: "session_start", SessionID: sid})
	if drop {
		t.Fatal("session_start should still dispatch for focus capture")
	}
	if store.Active(sid) {
		t.Fatal("session_start should reset subagent count")
	}
}
