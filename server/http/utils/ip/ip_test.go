package ip

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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
}

func TestIPParser(t *testing.T) {
	for i, test := range tests {
		t.Run("TestIPParser"+strconv.Itoa(i), func(t *testing.T) {
			err := ValidateAddress(test.IP)
			assert.Equal(t, test.Expected, err != nil, test.IP, err)
		})
	}
}

func TestParsePort(t *testing.T) {
	_, err := ParsePort(":8080")
	assert.NoError(t, err)
	_, err = ParsePort(":abc")
	assert.Error(t, err)
}
