package hertzh2c

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

	return tests.NewSuite(pluginsRoot, []string{"hertzhttp"})
}

func TestSuite(t *testing.T) {
	// Run the tests.
	suite.Run(t, newSuite())
}

func BenchmarkHTTPProto16(b *testing.B) {
	newSuite().Benchmark(b, "application/x-protobuf", 16)
}

func BenchmarkHTTPJSON16(b *testing.B) {
	newSuite().Benchmark(b, "application/json", 16)
}

func BenchmarkHTTPProto32(b *testing.B) {
	newSuite().Benchmark(b, "application/x-protobuf", 32)
}

func BenchmarkHTTPJSON32(b *testing.B) {
	newSuite().Benchmark(b, "application/json", 32)
}

func BenchmarkHTTPProto64(b *testing.B) {
	newSuite().Benchmark(b, "application/x-protobuf", 64)
}

func BenchmarkHTTPJSON64(b *testing.B) {
	newSuite().Benchmark(b, "application/json", 64)
}
