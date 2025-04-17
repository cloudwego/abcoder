package localsession

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
// SESSION_CONFIG_KEY is the env key for configuring default session manager.
//
//	Value format: [EnableImplicitlyTransmitAsync][,ShardNumber][,GCInterval]
//	- EnableImplicitlyTransmitAsync: 'true' means enabled, otherwist means disabled
//	- ShardNumber: integer > 0
//	- GCInterval: Golang time.Duration format, such as '10m' means ten minutes for each GC
//
// Once the key is set, default option values will be set if the option value doesn't exist.
const SESSION_CONFIG_KEY = "CLOUDWEGO_SESSION_CONFIG_KEY"

var defaultManagerObj *SessionManager

var defaultManagerOnce sync.Once

// DefaultManagerOptions returns default options for the default manager
func DefaultManagerOptions() ManagerOptions {
	return ManagerOptions{
		ShardNumber:                   100,
		GCInterval:                    time.Minute * 10,
		EnableImplicitlyTransmitAsync: false,
	}
}

// GetDefaultManager returns the default manager
func GetDefaultManager() *SessionManager {
	return defaultManagerObj
}

// InitDefaultManager update and restart default manager.
// It accept argument opts and env config both.
//
// NOTICE:
//   - It use env SESSION_CONFIG_KEY prior to argument opts;
//   - If both env and opts are empty, it won't reset manager;
//   - For concurrent safety, you can only successfully reset manager ONCE.
//
//go:nocheckptr
func InitDefaultManager(opts ManagerOptions) {
	defaultManagerOnce.Do(func() {
		// env config has high priority
		checkEnvOptions(&opts)

		if defaultManagerObj != nil {
			defaultManagerObj.Close()
		}
		obj := NewSessionManager(opts)
		defaultManagerObj = &obj
	})
}

func checkEnvOptions(opts *ManagerOptions) {
	if env := os.Getenv(SESSION_CONFIG_KEY); env != "" {
		envs := strings.Split(env, ",")
		// parse first option as EnableTransparentTransmitAsync
		if strings.ToLower(envs[0]) == "true" {
			opts.EnableImplicitlyTransmitAsync = true
		}

		// parse first option as ShardNumber
		if len(envs) > 1 {
			if opt, err := strconv.Atoi(envs[1]); err == nil {
				opts.ShardNumber = opt
			}
		}

		// parse third option as EnableTransparentTransmitAsync
		if len(envs) > 2 {
			if d, err := time.ParseDuration(envs[2]); err == nil && d > time.Second {
				opts.GCInterval = d
			}
		}
	}
}

// CurSession gets the session for current goroutine
//
// NOTICE: MUST call `InitDefaultManager()` once before using this API
func CurSession() (Session, bool) {
	if defaultManagerObj == nil {
		return nil, false
	}
	s, ok := defaultManagerObj.GetSession(SessionID(goID()))
	return s, ok
}

// BindSession binds the session with current goroutine
//
// NOTICE: MUST call `InitDefaultManager()` once before using this API
func BindSession(s Session) {
	if defaultManagerObj == nil {
		return
	}
	defaultManagerObj.BindSession(SessionID(goID()), s)
}

// UnbindSession unbind a session (if any) with current goroutine
//
// NOTICE: If you want to end the session,
// please call `Disable()` (or whatever make the session invalid)
// on your session's implementation
//
// NOTICE: MUST call `InitDefaultManager()` once before using this API
func UnbindSession() {
	if defaultManagerObj == nil {
		return
	}
	defaultManagerObj.UnbindSession(SessionID(goID()))
}

// Go calls f asynchronously and pass caller's session to the new goroutine
func Go(f func()) {
	s, ok := CurSession()
	if !ok {
		GoSession(nil, f)
	} else {
		GoSession(s, f)
	}
}

// SessionGo calls f asynchronously and pass s session to the new goroutine
func GoSession(s Session, f func()) {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				println(fmt.Sprintf("GoSession recover: %v", v))
			}
			UnbindSession()
		}()
		if s != nil {
			BindSession(s)
		}
		f()
	}()
}

