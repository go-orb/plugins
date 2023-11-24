package ip

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

var tests = []struct {
	IP       string
	Expected bool
}{
	{":8080", false},
	{"192.168.1.1:8080", false},
	{"500.168.1.1:8080", true},
	{"[::]:8080", false},
	{"localhost:8080", false},
	{"", true},
	{"8080", true},
	{"192.168.1.1:808080808080", true},
	{"8080:", true},
	{"", true},
	{":abc", true},
	{"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080", false},
	{"[2001:db8::1]:8080", false},
}

func TestIPParser(t *testing.T) {
	for i, test := range tests {
		t.Run("TestIPParser"+strconv.Itoa(i), func(t *testing.T) {
			err := ValidateAddress(test.IP)
			require.NoError(t, err)
			require.Equal(t, test.Expected, test.IP)
		})
	}
}

func TestParsePort(t *testing.T) {
	_, err := ParsePort(":8080")
	require.NoError(t, err)
	_, err = ParsePort(":abc")
	require.Error(t, err)
}
