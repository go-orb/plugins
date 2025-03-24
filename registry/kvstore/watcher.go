package kvstore

import (
	"errors"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/util/orberrors"
)

// Watcher is used to keep track of changes in the registry.
type Watcher struct {
	updates <-chan kvstore.WatchEvent
	stop    func() error

	codec            codecs.Marshaler
	database         string
	table            string
	serviceDelimiter string
}

// NewWatcher returns a new watcher.
func NewWatcher(s *Registry) (*Watcher, error) {
	w, ok := s.kvstore.KVStore.(kvstore.Watcher)
	if !ok {
		return nil, orberrors.ErrBadRequest.Wrap(errors.New("store does not implement watcher interface"))
	}

	watcher, stop, err := w.Watch(s.ctx, s.config.Database, s.config.Table)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		updates:          watcher,
		stop:             stop,
		codec:            s.codec,
		database:         s.config.Database,
		table:            s.config.Table,
		serviceDelimiter: s.config.ServiceDelimiter,
	}, nil
}

// Next returns the next result. It is a blocking call.
func (w *Watcher) Next() (*registry.Result, error) {
	kve := <-w.updates
	if kve.Key == "" {
		return nil, orberrors.ErrInternalServerError.Wrap(errors.New("watcher stopped"))
	}

	var svc registry.ServiceNode

	if kve.Value == nil {
		var err error

		svc, err = keyToServiceNode(kve.Key, w.serviceDelimiter)
		if err != nil {
			_ = w.stop() //nolint:errcheck
			return nil, orberrors.ErrBadRequest.Wrap(err)
		}
	} else {
		if err := w.codec.Unmarshal(kve.Value, &svc); err != nil {
			_ = w.stop() //nolint:errcheck
			return nil, orberrors.ErrInternalServerError.Wrap(err)
		}
	}

	var action registry.EventType

	switch kve.Operation {
	case kvstore.WatchOpCreate:
		action = registry.Create
	case kvstore.WatchOpUpdate:
		action = registry.Update
	case kvstore.WatchOpDelete:
		action = registry.Delete
	default:
		_ = w.stop() //nolint:errcheck
		return nil, orberrors.ErrBadRequest.Wrap(errors.New("invalid operation"))
	}

	return &registry.Result{
		Node:   svc,
		Action: action,
	}, nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	if w != nil {
		return w.stop()
	}

	return nil
}
