package grpc

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-orb/plugins/client/tests"
	"github.com/stretchr/testify/suite"

	o "github.com/go-orb/plugins/client/orb_transport/grpc"
)

func newSuite() *tests.TestSuite {
	_, filename, _, _ := runtime.Caller(0)
	pluginsRoot := filepath.Join(filepath.Dir(filename), "../../../../")

	s := tests.NewSuite(pluginsRoot, []string{o.Name})
	// s.Debug = true
	return s
}

func TestSuite(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	// Run the tests.
	suite.Run(t, newSuite())
}
