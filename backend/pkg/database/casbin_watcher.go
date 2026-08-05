package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2/persist"
	"github.com/redis/go-redis/v9"
)

const casbinPolicyReloadChannel = "pantheon:casbin:policy"

var (
	casbinWatcherMu sync.RWMutex
	casbinWatcher   persist.Watcher
)

func shouldEnableCasbinWatcher() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PANTHEON_CASBIN_WATCHER")), "true")
}

func initCasbinWatcher(client *redis.Client) persist.Watcher {
	if client == nil || !shouldEnableCasbinWatcher() {
		return nil
	}
	return newRedisCasbinWatcher(client)
}

func setCasbinWatcher(watcher persist.Watcher) {
	casbinWatcherMu.Lock()
	defer casbinWatcherMu.Unlock()

	if casbinWatcher != nil && casbinWatcher != watcher {
		casbinWatcher.Close()
	}
	casbinWatcher = watcher
}

// NotifyCasbinWatcher publishes a policy reload signal to other instances.
func NotifyCasbinWatcher() {
	casbinWatcherMu.RLock()
	watcher := casbinWatcher
	casbinWatcherMu.RUnlock()
	if watcher == nil {
		return
	}
	if err := watcher.Update(); err != nil {
		slog.Warn("casbin watcher update failed", "error", err)
	}
}

type redisCasbinWatcher struct {
	client     *redis.Client
	channel    string
	instanceID string

	callbackMu sync.RWMutex
	callback   func(string)

	stateMu sync.Mutex
	pubsub  *redis.PubSub

	cancel context.CancelFunc

	closeOnce sync.Once
}

func newRedisCasbinWatcher(client *redis.Client) persist.Watcher {
	if client == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	watcher := &redisCasbinWatcher{
		client:     client,
		channel:    casbinPolicyReloadChannel,
		instanceID: fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano()),
		cancel:     cancel,
	}
	go watcher.run(ctx)
	return watcher
}

func (w *redisCasbinWatcher) run(ctx context.Context) {
	pubsub := w.client.Subscribe(ctx, w.channel)
	w.stateMu.Lock()
	w.pubsub = pubsub
	w.stateMu.Unlock()
	defer func() {
		_ = pubsub.Close()
	}()

	if _, err := pubsub.Receive(ctx); err != nil {
		if !errorsIsContextCanceled(err) {
			slog.Warn("casbin watcher subscription failed", "error", err)
		}
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return
			}
			if strings.TrimSpace(msg.Payload) == "" || msg.Payload == w.instanceID {
				continue
			}
			callback := w.getCallback()
			if callback != nil {
				callback(msg.Payload)
			}
		}
	}
}

func (w *redisCasbinWatcher) getCallback() func(string) {
	w.callbackMu.RLock()
	defer w.callbackMu.RUnlock()
	return w.callback
}

func (w *redisCasbinWatcher) SetUpdateCallback(callback func(string)) error {
	w.callbackMu.Lock()
	w.callback = callback
	w.callbackMu.Unlock()
	return nil
}

func (w *redisCasbinWatcher) Update() error {
	return w.client.Publish(context.Background(), w.channel, w.instanceID).Err()
}

func (w *redisCasbinWatcher) Close() {
	w.closeOnce.Do(func() {
		w.cancel()
		w.stateMu.Lock()
		if w.pubsub != nil {
			_ = w.pubsub.Close()
			w.pubsub = nil
		}
		w.stateMu.Unlock()
	})
}

func errorsIsContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}
