package localsession

import (
	"sync"
	"sync/atomic"
	"time"
)
// ManagerOptions for SessionManager
type ManagerOptions struct {
	// EnableImplicitlyTransmitAsync enables transparently transmit
	// current session to children goroutines
	//
	// WARNING: Once this option enables, if you want to use `pprof.Do()`, it must be called before `BindSession()`,
	// otherwise transmitting will be disfunctional
	EnableImplicitlyTransmitAsync bool

	// ShardNumber is used to shard session id, it must be larger than zero
	ShardNumber int

	// GCInterval decides the GC interval for SessionManager,
	// it must be larger than 1s or zero means disable GC
	GCInterval time.Duration
}

type shard struct {
	lock sync.RWMutex
	m    map[SessionID]Session
}

// SessionManager maintain and manage sessions
type SessionManager struct {
	shards []*shard
	inGC   uint32
	tik    *time.Ticker
	opts   ManagerOptions
}

var defaultShardCap int = 10

func newShard() *shard {
	ret := new(shard)
	ret.m = make(map[SessionID]Session, defaultShardCap)
	return ret
}

// NewSessionManager creates a SessionManager with default containers
// If opts.GCInterval > 0, it will start scheduled GC() loop automatically
func NewSessionManager(opts ManagerOptions) SessionManager {
	if opts.ShardNumber <= 0 {
		panic("ShardNumber must be larger than zero")
	}
	shards := make([]*shard, opts.ShardNumber)
	for i := range shards {
		shards[i] = newShard()
	}
	ret := SessionManager{
		shards: shards,
		opts:   opts,
	}

	if opts.GCInterval > 0 {
		ret.startGC()
	}
	return ret
}

// Options shows the manager's Options
func (self SessionManager) Options() ManagerOptions {
	return self.opts
}

// SessionID is the identity of a session
type SessionID uint64

func (s *shard) Load(id SessionID) (Session, bool) {
	s.lock.RLock()

	// p := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.m)))
	// m := *(*map[SessionID]Session)(unsafe.Pointer(p))

	session, ok := s.m[id]
	s.lock.RUnlock()
	return session, ok
}

func (s *shard) Store(id SessionID, se Session) {
	s.lock.Lock()
	s.m[id] = se
	s.lock.Unlock()
}

func (s *shard) Delete(id SessionID) {
	s.lock.Lock()
	delete(s.m, id)
	s.lock.Unlock()
}

// Get gets specific session
// or get inherited session if option EnableImplicitlyTransmitAsync is true
func (self *SessionManager) GetSession(id SessionID) (Session, bool) {
	shard := self.shards[uint64(id)%uint64(self.opts.ShardNumber)]
	session, ok := shard.Load(id)
	if ok {
		return session, ok
	}
	if !self.opts.EnableImplicitlyTransmitAsync {
		return nil, false
	}

	id, ok = getSessionID()
	if !ok {
		return nil, false
	}
	shard = self.shards[uint64(id)%uint64(self.opts.ShardNumber)]
	return shard.Load(id)
}

// BindSession binds the session with current goroutine
func (self *SessionManager) BindSession(id SessionID, s Session) {
	shard := self.shards[uint64(id)%uint64(self.opts.ShardNumber)]

	shard.Store(id, s)

	if self.opts.EnableImplicitlyTransmitAsync {
		transmitSessionID(id)
	}
}

// UnbindSession clears current session
//
// Notice: If you want to end the session,
// please call `Disable()` (or whatever make the session invalid)
// on your session's implementation
func (self *SessionManager) UnbindSession(id SessionID) {
	shard := self.shards[uint64(id)%uint64(self.opts.ShardNumber)]

	_, ok := shard.Load(id)
	if ok {
		shard.Delete(id)
	}

	if self.opts.EnableImplicitlyTransmitAsync {
		clearSessionID()
	}
}

// GC sweep invalid sessions and release unused memory
func (self SessionManager) GC() {
	if !atomic.CompareAndSwapUint32(&self.inGC, 0, 1) {
		return
	}

	for _, shard := range self.shards {
		shard.lock.Lock()
		n := shard.m
		m := make(map[SessionID]Session, len(n))
		for id, s := range n {
			// Warning: may panic here?
			if s.IsValid() {
				m[id] = s
			}
		}
		// atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&shard.m)), unsafe.Pointer(&m))
		shard.m = m
		shard.lock.Unlock()
	}

	atomic.StoreUint32(&self.inGC, 0)
}

// startGC start a scheduled goroutine to call GC() according to GCInterval
func (self *SessionManager) startGC() {
	if self.opts.GCInterval < time.Second {
		panic("GCInterval must be larger than 1 second")
	}
	self.tik = time.NewTicker(self.opts.GCInterval)
	go func() {
		for range self.tik.C {
			self.GC()
		}
	}()
}

// Close stop persistent work for the manager, like GC
func (self SessionManager) Close() {
	if self.tik != nil {
		self.tik.Stop()
	}
}

