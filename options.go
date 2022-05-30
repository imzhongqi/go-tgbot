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

// PanicHandler is panic handler
type PanicHandler func(*Context, interface{})

// PollUpdatesErrorHandler is the handler that is called when an error occurs in the polling updates
type PollUpdatesErrorHandler func(err error)

type Options struct {
	ctx context.Context

	// timeout is context timeout.
	timeout time.Duration

	// autoSetupCommands whether automatically set up commands.
	autoSetupCommands bool

	undefinedCommandHandler Handler
	errHandler              ErrHandler
	updatesHandler          UpdatesHandler
	panicHandler            PanicHandler
	pollUpdatesErrorHandler PollUpdatesErrorHandler

	workerNum  int
	workerPool *ants.Pool

	// bufSize is updateC chan buffer size.
	bufSize int

	updateTimeout  int
	limit          int
	offset         int
	allowedUpdates []string
}

func newOptions(opts ...Option) *Options {
	options := &Options{
		ctx: context.Background(),

		autoSetupCommands: true,
		errHandler:        func(err error) {},

		workerNum: runtime.GOMAXPROCS(0),

		updateTimeout: 50, // 50s is maximum timeout
		limit:         100,
	}

	options.panicHandler = func(ctx *Context, v interface{}) {
		options.errHandler(fmt.Errorf("tgbot panic: %v, stack: %s", v, debug.Stack()))
	}

	options.pollUpdatesErrorHandler = func(err error) {
		options.errHandler(fmt.Errorf("failed to get updates, error: %w", err))
		time.Sleep(3 * time.Second)
	}

	for _, o := range opts {
		o(options)
	}

	return options
}

type Option func(b *Options)

// WithTimeout set context timeout.
func WithTimeout(d time.Duration) Option {
	return func(b *Options) {
		b.timeout = d
	}
}

// WithUpdateTimeout set the get updates updateTimeout,
// timeout unit is seconds, max is 50 second.
func WithUpdateTimeout(timeout int) Option {
	return func(b *Options) {
		b.updateTimeout = timeout
	}
}

// WithWorkerNum set the number of workers to process updates.
func WithWorkerNum(n int) Option {
	return func(b *Options) {
		if b.workerNum > 0 {
			b.workerNum = n
		}
	}
}

// WithWorkerPool set the worker pool for execute handler if the workerPool is non-nil.
func WithWorkerPool(p *ants.Pool) Option {
	return func(b *Options) {
		b.workerPool = p
	}
}

// WithUndefinedCmdHandler set how to handle undefined commands.
func WithUndefinedCmdHandler(h Handler) Option {
	return func(b *Options) {
		b.undefinedCommandHandler = h
	}
}

// WithErrorHandler set error handler.
func WithErrorHandler(h ErrHandler) Option {
	return func(b *Options) {
		b.errHandler = h
	}
}

// WithAutoSetupCommands will auto setup command to telegram if true.
func WithAutoSetupCommands(v bool) Option {
	return func(b *Options) {
		b.autoSetupCommands = v
	}
}

// WithBufferSize set the buffer size for receive updates.
func WithBufferSize(size int) Option {
	return func(b *Options) {
		b.bufSize = size
	}
}

// WithLimitUpdates set the get updates limit.
func WithLimitUpdates(limit int) Option {
	return func(b *Options) {
		b.limit = limit
	}
}

// WithUpdatesHandler set the updates handler.
func WithUpdatesHandler(handler UpdatesHandler) Option {
	return func(b *Options) {
		b.updatesHandler = handler
	}
}

// WithPanicHandler set panic handler.
func WithPanicHandler(h PanicHandler) Option {
	return func(b *Options) {
		b.panicHandler = h
	}
}

// WithAllowedUpdates set allowed updates.
func WithAllowedUpdates(v ...string) Option {
	return func(b *Options) {
		b.allowedUpdates = v
	}
}

// WithContext with the context.
func WithContext(ctx context.Context) Option {
	return func(b *Options) {
		b.ctx = ctx
	}
}
