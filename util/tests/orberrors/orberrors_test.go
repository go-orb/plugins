package orberrors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/stretchr/testify/require"
)

func TestErrorIs(t *testing.T) {
	err := orberrors.HTTP(http.StatusUnauthorized)
	require.ErrorIs(t, err, orberrors.ErrUnauthorized)
}

func TestErrorIsWrapped(t *testing.T) {
	err := orberrors.HTTP(http.StatusUnauthorized).Wrap(errors.New("test"))
	require.ErrorIs(t, err, orberrors.ErrUnauthorized)
}

func TestError(t *testing.T) {
	msg := fmt.Sprintf("%v", orberrors.ErrInternalServerError)
	expected := "internal server error"
	require.Equal(t, expected, msg)
}

func TestWrappedError(t *testing.T) {
	err := orberrors.ErrInternalServerError.Wrap(errors.New("testing"))
	expected := "internal server error: testing"
	require.Equal(t, expected, err.Error())
}

func TestNew(t *testing.T) {
	msg := orberrors.New(500, "testing").Error()
	expected := "testing"
	require.Equal(t, expected, msg)
}

func TestNewHTTP(t *testing.T) {
	msg := orberrors.NewHTTP(500).Error()
	expected := "internal server error"
	require.Equal(t, expected, msg)
}

func TestFrom(t *testing.T) {
	msg := orberrors.From(errors.New("testing")).Error()
	expected := "internal server error: testing"
	require.Equal(t, expected, msg)
}

func TestAs(t *testing.T) {
	orbe, ok := orberrors.As(orberrors.ErrRequestTimeout)
	require.True(t, ok)
	require.Equal(t, 408, orbe.Code)
}

func TestFromAndAs(t *testing.T) {
	err := orberrors.From(errors.New("testing"))
	orbe, ok := orberrors.As(err)
	require.True(t, ok)
	require.Equal(t, 500, orbe.Code)
}

func TestWrappedAs(t *testing.T) {
	err := orberrors.ErrRequestTimeout.Wrap(errors.New("test"))
	require.Equal(t, "request timeout: test", err.Error())
	orbe, ok := orberrors.As(err)
	require.True(t, ok)
	require.Equal(t, "request timeout: test", orbe.Error())
	require.Equal(t, 408, orbe.Code)
}
