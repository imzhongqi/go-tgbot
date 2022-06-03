package tgbot

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/panjf2000/ants/v2"
)

// UpdatesHandler handler another update.
type UpdatesHandler func(ctx *Context)

// Handler command handler.
type Handler func(ctx *Context) error

// ErrHandler error handler.
type ErrHandler func(err error)

// PanicHandler is panic handler.
type PanicHandler func(*Context, interface{})

type options struct {
	ctx context.Context

	// timeout is context timeout.
	timeout time.Duration

	// disableAutoSetupCommands whether automatically set up commands.
	disableAutoSetupCommands bool

	disableHandleAllUpdateOnStop bool

	undefinedCommandHandler Handler
	errHandler              ErrHandler
	updatesHandler          UpdatesHandler
	panicHandler            PanicHandler

	// pollUpdatesErrorHandler is the handler that is called when an error occurs in the polling updates.
	pollUpdatesErrorHandler ErrHandler

	workerNum  int
	workerPool *ants.Pool

	// bufSize is updateC chan buffer size.
	bufSize int

	updateTimeout  int
	limit          int
	offset         int
	allowedUpdates []string
}

func newOptions(opts ...Option) *options {
	o := &options{
		ctx: context.Background(),

		errHandler: func(err error) {},

		workerNum: runtime.GOMAXPROCS(0),

		updateTimeout: 50, // 50s is maximum timeout.
		limit:         100,
	}

	o.panicHandler = func(ctx *Context, v interface{}) {
		o.errHandler(fmt.Errorf("tgbot panic: %v, stack: %s", v, debug.Stack()))
	}

	o.pollUpdatesErrorHandler = func(err error) {
		o.errHandler(fmt.Errorf("failed to get updates, error: %w", err))
		time.Sleep(3 * time.Second)
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

type Option func(o *options)

// WithContext with the context.
func WithContext(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
	}
}

// WithTimeout set context timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// WithWorkerNum set the number of workers to process updates.
func WithWorkerNum(n int) Option {
	return func(o *options) {
		o.workerNum = n
	}
}

// WithWorkerPool set the worker pool for execute handler if the workerPool is non-nil.
func WithWorkerPool(p *ants.Pool) Option {
	return func(o *options) {
		o.workerPool = p
	}
}

// WithUndefinedCmdHandler set how to handle undefined commands.
func WithUndefinedCmdHandler(h Handler) Option {
	return func(o *options) {
		o.undefinedCommandHandler = h
	}
}

// WithErrorHandler set error handler.
func WithErrorHandler(h ErrHandler) Option {
	return func(o *options) {
		o.errHandler = h
	}
}

// WithDisableAutoSetupCommands disable auto setup telegram commands.
func WithDisableAutoSetupCommands(v bool) Option {
	return func(o *options) {
		o.disableAutoSetupCommands = v
	}
}

// WithDisableHandleAllUpdateOnStop disable handle all updates on stop.
func WithDisableHandleAllUpdateOnStop(v bool) Option {
	return func(o *options) {
		o.disableHandleAllUpdateOnStop = v
	}
}

// WithBufferSize set the buffer size for receive updates.
func WithBufferSize(size int) Option {
	return func(o *options) {
		o.bufSize = size
	}
}

// WithUpdatesHandler set the updates handler.
func WithUpdatesHandler(handler UpdatesHandler) Option {
	return func(o *options) {
		o.updatesHandler = handler
	}
}

// WithPanicHandler set panic handler.
func WithPanicHandler(h PanicHandler) Option {
	return func(o *options) {
		o.panicHandler = h
	}
}

// WithGetUpdatesTimeout set the get updates updateTimeout,
// timeout unit is seconds, max is 50 second.
func WithGetUpdatesTimeout(timeout int) Option {
	return func(o *options) {
		o.updateTimeout = timeout
	}
}

// WithGetUpdatesLimit set the get updates limit.
func WithGetUpdatesLimit(limit int) Option {
	return func(o *options) {
		o.limit = limit
	}
}

// WithGetUpdatesAllowedUpdates set allowed updates.
func WithGetUpdatesAllowedUpdates(v ...string) Option {
	return func(o *options) {
		o.allowedUpdates = v
	}
}
