package tgbot

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewOptions(t *testing.T) {
	ctx := context.Background()
	timeout := 10 * time.Second
	updateTimeout := 30
	workerNum := 10
	bufferSize := 20
	allowedUpdates := []string{"ok"}
	limit := 20

	opts := []Option{
		WithContext(ctx),
		WithDisableAutoSetupCommands(true),
		WithDisableHandleAllUpdateOnStop(true),
		WithTimeout(timeout),
		WithGetUpdatesTimeout(updateTimeout),
		WithWorkersNum(workerNum),
		WithUndefinedCmdHandler(nil),
		WithBufferSize(bufferSize),
		WithGetUpdatesAllowedUpdates(allowedUpdates...),
		WithPanicHandler(nil),
		WithGetUpdatesLimit(limit),
	}

	o := newOptions(opts...)

	if o.ctx != ctx {
		t.Errorf("o.ctx except %v, got: %v", ctx, o.ctx)
	}

	if o.disableAutoSetupCommands != true {
		t.Errorf("o.disableAutoSetupCommands except %v, got: %v", true, o.disableAutoSetupCommands)
	}

	if o.disableHandleAllUpdateOnStop != true {
		t.Errorf("o.disableHandleAllUpdateOnStop except %v, got: %v", true, o.disableHandleAllUpdateOnStop)
	}

	if o.timeout != timeout {
		t.Errorf("o.timeout except %v, got: %v", timeout, o.timeout)
	}

	if o.updateTimeout != updateTimeout {
		t.Errorf("o.updateTimeout except %v, got: %v", updateTimeout, o.updateTimeout)
	}

	if o.workersNum != workerNum {
		t.Errorf("o.workersNum except %v, got: %v", workerNum, o.workersNum)
	}

	if o.undefinedCommandHandler != nil {
		t.Errorf("o.undefinedCommandHandler except %v, got: %p", nil, o.undefinedCommandHandler)
	}

	if o.bufSize != bufferSize {
		t.Errorf("o.bufSize except %v, got: %v", bufferSize, o.bufSize)
	}

	if !reflect.DeepEqual(o.allowedUpdates, allowedUpdates) {
		t.Errorf("o.allowedUpdates except %v, got: %v", allowedUpdates, o.allowedUpdates)
	}

	if o.panicHandler != nil {
		t.Errorf("o.panicHandler except %v, got: %p", nil, o.panicHandler)
	}

	if o.limit != limit {
		t.Errorf("o.limit except %v, got: %v", limit, o.limit)
	}
}
