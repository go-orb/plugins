package router

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-orb/go-orb/util/container"
	"gopkg.in/yaml.v3"
)

// Middleware contains a map of middlewares, by name. These can by
// referenced dynamically from your config files.
var Middleware = container.NewPlugins[func(http.Handler) http.Handler]() //nolint:gochecknoglobals

// Middleware is a wrapper around the func(http.Handler) http.Handler convention
// used for middleware. We use this alias to implement custom marshal/unmarshal
// methods for the config, to allow you to dynamcially change them.
// type Middleware func(http.Handler) http.Handler

// MarshalJSON no-op.
func (m Middlewares) MarshalJSON() ([]byte, error) {
	return nil, nil
}

// MarshalYAML no-op.
func (m Middlewares) MarshalYAML() ([]byte, error) {
	return nil, nil
}

// UnmarshalText middleware.
func (m Middlewares) UnmarshalText(data []byte) error {
	return m.UnmarshalJSON(data)
}

// UnmarshalJSON middlware.
func (m Middlewares) UnmarshalJSON(data []byte) error {
	var middlewares []string

	if err := json.Unmarshal(data, &middlewares); err != nil {
		return err
	}

	return m.set(middlewares)
}

// UnmarshalYAML middlware list.
func (m Middlewares) UnmarshalYAML(data *yaml.Node) error {
	var middlewares []string

	if err := data.Decode(&middlewares); err != nil {
		return err
	}

	return m.set(middlewares)
}

func (m Middlewares) set(middlewares []string) error {
	for _, name := range middlewares {
		middleware, ok := Middleware.Get(name)
		if !ok {
			return fmt.Errorf("middleware %s not found, did you register it?", name)
		}

		m[name] = middleware
	}

	return nil
}
