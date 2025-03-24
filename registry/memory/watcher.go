package memory

import (
	"context"
	"errors"

	"github.com/go-orb/go-orb/registry"
)

type watcher struct {
	ctx context.Context

	wo  registry.WatchOptions
	res chan *registry.Result
	id  string
}

func (m *watcher) Next() (*registry.Result, error) {
	for {
		select {
		case r := <-m.res:
			if len(m.wo.Service) > 0 && m.wo.Service != r.Node.Name {
				continue
			}

			return r, nil
		case <-m.ctx.Done():
			return nil, errors.New("watcher stopped")
		}
	}
}
