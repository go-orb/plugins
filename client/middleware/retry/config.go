package retry

import "github.com/go-orb/go-orb/client"

//nolint:gochecknoglobals
var (
	// DefaultFunc is the default retry function.
	// note that returning either false or a non-nil error will result in the call not being retried.
	DefaultFunc = OnConnectionError

	// DefaultRetries is the default number of times a request is tried.
	// Set it to 0 to disable retries.
	DefaultRetries = 5
)

// Config is the retry middleware config.
type Config struct {
	// RetryFunc is the retry function.
	// Default is OnConnectionError.
	RetryFunc client.RetryFunc `json:"-" yaml:"-"`

	// Retries is the number of times a request is tried.
	// Set it to 0 to disable retries.
	// Default is 5.
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// NewConfig returns a new config object.
func NewConfig() Config {
	cfg := Config{
		RetryFunc: DefaultFunc,
		Retries:   DefaultRetries,
	}

	return cfg
}
