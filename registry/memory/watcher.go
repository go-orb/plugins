package memory

import (
	"errors"

	"github.com/go-orb/go-orb/registry"
)

type watcher struct {
	wo   registry.WatchOptions
	res  chan *registry.Result
	exit chan bool
	id   string
}

func (m *watcher) Next() (*registry.Result, error) {
	for {
		select {
		case r := <-m.res:
			if len(m.wo.Service) > 0 && m.wo.Service != r.Service.Name {
				continue
			}

			return r, nil
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *watcher) Stop() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}

	return nil
}
