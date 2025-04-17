package localsession

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)
// Session represents a local storage for one session
type Session interface {
	// IsValid tells if the session is valid at present
	IsValid() bool

	// Get returns value for specific key
	Get(key interface{}) interface{}

	// WithValue sets value for specific key，and return newly effective session
	WithValue(key interface{}, val interface{}) Session
}

// SessionCtx implements Session with context,
// which means children session WON'T affect parent and sibling sessions
type SessionCtx struct {
	enabled *atomic.Value
	storage context.Context
}

// NewSessionCtx creates and enables a SessionCtx
func NewSessionCtx(ctx context.Context) SessionCtx {
	var enabled atomic.Value
	enabled.Store(true)
	return SessionCtx{
		enabled: &enabled,
		storage: ctx,
	}
}

// NewSessionCtx creates and enables a SessionCtx,
// and disable the session after timeout
func NewSessionCtxWithTimeout(ctx context.Context, timeout time.Duration) SessionCtx {
	ret := NewSessionCtx(ctx)
	go func() {
		<-time.NewTimer(timeout).C
		ret.Disable()
	}()
	return ret
}

// Disable ends the session
func (self SessionCtx) Disable() {
	self.enabled.Store(false)
}

// Export exports underlying context
func (self SessionCtx) Export() context.Context {
	return self.storage
}

// IsValid tells if the session is valid at present
func (self SessionCtx) IsValid() bool {
	return self.enabled.Load().(bool)
}

// Get value for specific key
func (self SessionCtx) Get(key interface{}) interface{} {
	return self.storage.Value(key)
}

// Set value for specific key，and return newly effective session
func (self SessionCtx) WithValue(key interface{}, val interface{}) Session {
	ctx := context.WithValue(self.storage, key, val)
	return SessionCtx{
		enabled: self.enabled,
		storage: ctx,
	}
}

// NewSessionMap implements Session with map,
// which means children session WILL affect parent session and sibling sessions
type SessionMap struct {
	enabled *atomic.Value
	storage map[interface{}]interface{}
	lock    sync.RWMutex
}

// NewSessionMap creates and enables a SessionMap
func NewSessionMap(m map[interface{}]interface{}) *SessionMap {
	var enabled atomic.Value
	enabled.Store(true)
	return &SessionMap{
		enabled: &enabled,
		storage: m,
	}
}

// NewSessionCtx creates and enables a SessionCtx,
// and disable the session after timeout
func NewSessionMapWithTimeout(m map[interface{}]interface{}, timeout time.Duration) *SessionMap {
	ret := NewSessionMap(m)
	go func() {
		<-time.NewTimer(timeout).C
		ret.Disable()
	}()
	return ret
}

// IsValid tells if the session is valid at present
func (self *SessionMap) IsValid() bool {
	if self == nil {
		return false
	}
	return self.enabled.Load().(bool)
}

// Disable ends the session
func (self *SessionMap) Disable() {
	if self == nil {
		return
	}
	self.enabled.Store(false)
}

// Export COPIES and exports underlying map
func (self *SessionMap) Export() map[interface{}]interface{} {
	if self == nil {
		return nil
	}
	m := make(map[interface{}]interface{}, len(self.storage))
	self.lock.RLock()
	for k, v := range self.storage {
		m[k] = v
	}
	self.lock.RUnlock()
	return m
}

// Get value for specific key
func (self *SessionMap) Get(key interface{}) interface{} {
	if self == nil {
		return nil
	}
	self.lock.RLock()
	val := self.storage[key]
	self.lock.RUnlock()
	return val
}

// Set value for specific key，and return itself
func (self *SessionMap) WithValue(key, val interface{}) Session {
	if self == nil {
		return nil
	}
	self.lock.Lock()
	self.storage[key] = val
	self.lock.Unlock()
	return self
}

