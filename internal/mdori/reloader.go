package mdori

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type reloader struct {
	watcher  *fsnotify.Watcher
	baseName string

	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
	closed      bool
	closeOnce   sync.Once
}

func newReloader(ctx context.Context, sourcePath string) (*reloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(sourcePath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, err
	}

	r := &reloader{
		watcher:     watcher,
		baseName:    filepath.Base(sourcePath),
		subscribers: make(map[chan struct{}]struct{}),
	}

	go r.run(ctx)

	return r, nil
}

func (r *reloader) run(ctx context.Context) {
	defer r.closeSubscribers()
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			if !r.matches(event) {
				continue
			}

			r.notify()
		case _, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (r *reloader) matches(event fsnotify.Event) bool {
	if filepath.Base(event.Name) != r.baseName {
		return false
	}

	return event.Has(fsnotify.Write) ||
		event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Rename) ||
		event.Has(fsnotify.Remove)
}

func (r *reloader) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		close(ch)
		return ch
	}

	r.subscribers[ch] = struct{}{}

	return ch
}

func (r *reloader) Unsubscribe(ch chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.subscribers[ch]; !ok {
		return
	}

	delete(r.subscribers, ch)
	close(ch)
}

func (r *reloader) notify() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ch := range r.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (r *reloader) Close() {
	r.closeOnce.Do(func() {
		_ = r.watcher.Close()
	})
}

func (r *reloader) closeSubscribers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}

	r.closed = true

	for ch := range r.subscribers {
		close(ch)
		delete(r.subscribers, ch)
	}
}
