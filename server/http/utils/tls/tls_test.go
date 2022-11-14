package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCertificate(t *testing.T) {
	_, err := GenTlSConfig("localhost:8080")
	assert.NoError(t, err)

	_, err = CertificateQuic()
	assert.NoError(t, err)
}
