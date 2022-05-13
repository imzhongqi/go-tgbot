package tgbot

import (
	"context"

	"github.com/panjf2000/ants/v2"
)

type Option func(b *Bot)

// WithTimeout set the get updates timeout.
func WithTimeout(timeout int) Option {
	return func(b *Bot) {
		b.timeout = timeout
	}
}

// WithWorkerNum set the number of workers to process updates.
func WithWorkerNum(n int) Option {
	return func(b *Bot) {
		if b.workerNum > 0 {
			b.workerNum = n
		}
	}
}

// WithWorkerPool set the worker pool for execute handler if the workerPool is non-nil.
func WithWorkerPool(p *ants.Pool) Option {
	return func(b *Bot) {
		b.workerPool = p
	}
}

// WithUndefinedCmdHandler set how to handle undefined commands.
func WithUndefinedCmdHandler(h Handler) Option {
	return func(b *Bot) {
		if h != nil {
			b.undefinedCommandHandler = h
		}
	}
}

// WithErrorHandler set error handler
func WithErrorHandler(h ErrHandler) Option {
	return func(b *Bot) {
		if h != nil {
			b.errHandler = h
		}
	}
}

// WithBufferSize set the buffer size for receive updates.
func WithBufferSize(size int) Option {
	return func(b *Bot) {
		b.bufSize = size
	}
}

// WithLimitUpdates set the get updates limit.
func WithLimitUpdates(limit int) Option {
	return func(b *Bot) {
		b.limit = limit
	}
}

// WithUpdatesHandler set the updates handler
func WithUpdatesHandler(handler UpdatesHandler) Option {
	return func(b *Bot) {
		b.updatesHandler = handler
	}
}

// WithPanicHandler set panic handler
func WithPanicHandler(h func(interface{}) string) Option {
	return func(b *Bot) {
		b.panicHandler = h
	}
}

// WithAllowedUpdates set allowed updates
func WithAllowedUpdates(v ...string) Option {
	return func(b *Bot) {
		b.allowedUpdates = v
	}
}

// WithContext with the context
func WithContext(ctx context.Context) Option {
	return func(b *Bot) {
		b.ctx = ctx
	}
}
