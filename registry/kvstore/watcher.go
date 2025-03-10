package kvstore

import (
	"errors"
	"strings"

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

	var svc registry.Service

	if kve.Value == nil {
		// fake a service
		parts := strings.SplitN(kve.Key, w.serviceDelimiter, 3)
		if len(parts) != 3 {
			return nil, orberrors.ErrBadRequest.Wrap(errors.New("invalid service key"))
		}

		svc.Name = parts[0]

		// go-orb registers nodes with a - separator
		svc.Nodes = []*registry.Node{{ID: parts[0] + "-" + parts[1]}}
		svc.Version = parts[2]
	} else {
		if err := w.codec.Unmarshal(kve.Value, &svc); err != nil {
			_ = w.stop() //nolint:errcheck
			return nil, orberrors.ErrInternalServerError.Wrap(err)
		}
	}

	actionName := ""

	switch kve.Operation {
	case kvstore.WatchOpCreate:
		actionName = "create"
	case kvstore.WatchOpUpdate:
		actionName = "update"
	case kvstore.WatchOpDelete:
		actionName = "delete"
	default:
		_ = w.stop() //nolint:errcheck
		return nil, orberrors.ErrBadRequest.Wrap(errors.New("invalid operation"))
	}

	return &registry.Result{
		Service: &svc,
		Action:  actionName,
	}, nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	return w.stop()
}
