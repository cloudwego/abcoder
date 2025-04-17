package backup

import (
	"context"
	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/localsession"
)
// BackupHandler is used to decide and recover prev context to cur context
type BackupHandler func(prev, cur context.Context) (ctx context.Context, backup bool)

// Options
type Options struct {
	Enable bool
	localsession.ManagerOptions
}

// Default Options
func DefaultOptions() Options {
	return Options{
		Enable:         false,
		ManagerOptions: localsession.DefaultManagerOptions(),
	}
}

// Init gloabal session manager
// It uses env config first, the key is localsession.SESSION_CONFIG_KEY
func Init(opts Options) {
	if opts.Enable {
		localsession.InitDefaultManager(opts.ManagerOptions)
	}
}

// If handler != nil, this func will try to merge metainfo
// and pre-defined key-values (through Options.BackupHanlder)
// from backup context into given context
func RecoverCtxOnDemands(ctx context.Context, handler BackupHandler) context.Context {
	if handler == nil {
		return ctx
	}
	s, ok := localsession.CurSession()
	if !ok {
		return ctx
	}
	c, ok := s.(localsession.SessionCtx)
	if !ok {
		return ctx
	}
	pre := c.Export()

	// trigger user-defined handler if any
	nctx, backup := handler(pre, ctx)
	if !backup {
		return ctx
	}
	ctx = nctx

	// two-way merge all persistent metainfo if pre context has
	if n := metainfo.CountPersistentValues(pre); n > 0 {
		// persistent kvs
		kvs := make([]string, 0, n*2)
		mkvs := metainfo.GetAllPersistentValues(ctx)

		// incoming ctx is prior to session
		if len(mkvs) == 0 {
			// merge all kvs from pre
			metainfo.RangePersistentValues(pre, func(k, v string) bool {
				kvs = append(kvs, k, v)
				return true
			})
		} else {
			metainfo.RangePersistentValues(pre, func(k, v string) bool {
				// filter kvs which exists in cur
				if _, ok := mkvs[k]; !ok {
					kvs = append(kvs, k, v)
				}
				return true
			})
		}
		ctx = metainfo.WithPersistentValues(ctx, kvs...)
	}

	return ctx
}

// Set current Sessioin
func BackupCtx(ctx context.Context) {
	if localsession.GetDefaultManager() == nil {
		return
	}
	localsession.BindSession(localsession.NewSessionCtx(ctx))
}

// Unset current Session
func ClearCtx() {
	localsession.UnbindSession()
}

