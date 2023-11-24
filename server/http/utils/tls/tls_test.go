package tls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateCertificate(t *testing.T) {
	_, err := GenTLSConfig("localhost:8080")
	require.NoError(t, err)

	_, err = CertificateQuic()
	require.NoError(t, err)
}
