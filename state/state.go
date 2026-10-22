// Package state holds ish's application state, such as its configuration,
// for the parts of ish that read it, like filters. Readers take a snapshot,
// and do all of their lookups on it, so a change to the state can't change
// values between their reads.
package state

import (
	"sync/atomic"

	"github.com/factory-factory/ish/config"
)

// State is the application state. Its methods are safe to call from any
// goroutine.
type State struct {
	current atomic.Pointer[Snapshot]
}

// Snapshot is the application state at one moment. It's never changed, so it
// can be read without locking; a change to the state makes a new snapshot.
type Snapshot struct {
	config config.Config
}

// New returns the application state, with the configuration cfg.
func New(cfg config.Config) *State {
	s := &State{}
	s.SetConfig(cfg)

	return s
}

// Snapshot returns the current state. Lookups on it give the same answers,
// however the state changes afterwards.
func (s *State) Snapshot() *Snapshot {
	if s == nil {
		return &Snapshot{config: config.Config{}}
	}

	if snap := s.current.Load(); snap != nil {
		return snap
	}

	return &Snapshot{config: config.Config{}}
}

// SetConfig replaces the configuration, as editing the settings would. Later
// snapshots have the new configuration; earlier ones keep the old.
func (s *State) SetConfig(cfg config.Config) {
	// A copy, so the caller can't change the snapshot's configuration.
	copied, _ := cfg.Lookup("")
	snapshot := config.Config{}

	if m, ok := copied.(map[string]any); ok {
		snapshot = config.Config(m)
	}

	s.current.Store(&Snapshot{config: snapshot})
}

// Lookup returns the configuration value at path, a dotted path such as
// .user.email. See config.Config.Lookup.
func (s *Snapshot) Lookup(path string) (any, bool) {
	return s.config.Lookup(path)
}

// LookupString returns the configuration value at path, if it's a string.
func (s *Snapshot) LookupString(path string) (string, bool) {
	v, ok := s.Lookup(path)
	if !ok {
		return "", false
	}

	str, ok := v.(string)

	return str, ok
}
