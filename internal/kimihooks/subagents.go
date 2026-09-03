package kimihooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
)

const subagentLockTimeout = 2 * time.Second

// subagentStore counts in-flight Kimi subagents per session.
// Kimi fires a plain Stop when a subagent turn ends, with the same payload as
// the main agent; that Stop arrives before SubagentStop. Dropping Stop while
// the count is > 0 avoids notifying "main agent done" for a child return.
type subagentStore struct {
	path string
	mu   sync.Mutex
}

type subagentFile struct {
	Sessions map[string]int `json:"sessions"`
}

func subagentStorePath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "kimi-subagents.json")
}

func newSubagentStore(path string) *subagentStore {
	return &subagentStore{path: path}
}

func (s *subagentStore) Begin(sessionID string) {
	s.delta(sessionID, 1)
}

func (s *subagentStore) End(sessionID string) {
	s.delta(sessionID, -1)
}

func (s *subagentStore) Reset(sessionID string) {
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := common.AcquireFileLock(s.path+".lock", subagentLockTimeout)
	if lock != nil {
		defer lock.Release()
	}
	sf := s.load()
	delete(sf.Sessions, sessionID)
	_ = s.save(sf)
}

func (s *subagentStore) Active(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load().Sessions[sessionID] > 0
}

func (s *subagentStore) delta(sessionID string, n int) {
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := common.AcquireFileLock(s.path+".lock", subagentLockTimeout)
	if lock != nil {
		defer lock.Release()
	}
	sf := s.load()
	next := sf.Sessions[sessionID] + n
	if next <= 0 {
		delete(sf.Sessions, sessionID)
	} else {
		sf.Sessions[sessionID] = next
	}
	_ = s.save(sf)
}

func (s *subagentStore) load() subagentFile {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return subagentFile{Sessions: map[string]int{}}
	}
	var sf subagentFile
	if len(data) > 0 {
		if err := json.Unmarshal(data, &sf); err != nil {
			return subagentFile{Sessions: map[string]int{}}
		}
	}
	if sf.Sessions == nil {
		sf.Sessions = map[string]int{}
	}
	return sf
}

func (s *subagentStore) save(sf subagentFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return common.WriteFileAtomic(s.path, data, 0o644)
}

// trackSubagentLifecycle updates the in-flight subagent count and reports
// whether this message should skip notification.
func trackSubagentLifecycle(store *subagentStore, msg notify.Message) (drop bool, reason string) {
	switch msg.Event {
	case "subagent_start":
		store.Begin(msg.SessionID)
		return true, "subagent_start"
	case "subagent_stop":
		store.End(msg.SessionID)
		return true, "subagent_stop"
	case "session_start":
		store.Reset(msg.SessionID)
		return false, ""
	case "run_completed":
		if store.Active(msg.SessionID) {
			return true, "skip stop while subagent running"
		}
	}
	return false, ""
}
