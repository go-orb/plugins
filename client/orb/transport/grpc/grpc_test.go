package grpc

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-orb/plugins/client/tests"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/jsonpb"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/registry/consul"
	_ "github.com/go-orb/plugins/registry/mdns"
	_ "github.com/go-orb/plugins/server/http/router/chi"
)

func newSuite() *tests.TestSuite {
	_, filename, _, _ := runtime.Caller(0)
	pluginsRoot := filepath.Join(filepath.Dir(filename), "../../../../")

	s := tests.NewSuite(pluginsRoot, []string{"grpc"})
	// s.Debug = true
	return s
}

func TestSuite(t *testing.T) {
	// Run the tests.
	suite.Run(t, newSuite())
}

func BenchmarkGRPC16(b *testing.B) {
	newSuite().Benchmark(b, "application/proto", 16)
}
