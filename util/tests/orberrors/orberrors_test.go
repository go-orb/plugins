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

func TestErrorIsInAnotherError(t *testing.T) {
	err := fmt.Errorf("Test: %w", orberrors.HTTP(http.StatusUnauthorized).Wrap(errors.New("test")))
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
	msg := orberrors.HTTP(500).Error()
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

func TestNilError(t *testing.T) {
	var err *orberrors.Error
	require.Equal(t, "", err.Error())
	require.NoError(t, err.Toerror())
	require.Error(t, err.Wrap(errors.New("test")))
	require.NoError(t, err.Unwrap())
}

func TestToerror(t *testing.T) {
	// Test non-nil case
	orbErr := orberrors.New(400, "bad request")
	err := orbErr.Toerror()
	require.Error(t, err)
	require.Equal(t, "bad request", err.Error())

	// Test nil case
	var nilErr *orberrors.Error
	require.NoError(t, nilErr.Toerror())
}

func TestHTTPWithDifferentCodes(t *testing.T) {
	// Test predefined HTTP codes
	testCases := []struct {
		code     int
		expected *orberrors.Error
	}{
		{503, orberrors.ErrUnavailable},
		{500, orberrors.ErrInternalServerError},
		{499, orberrors.ErrCanceled},
		{401, orberrors.ErrUnauthorized},
		{408, orberrors.ErrRequestTimeout},
		{400, orberrors.ErrBadRequest},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("HTTP_%d", tc.code), func(t *testing.T) {
			err := orberrors.HTTP(tc.code)
			require.ErrorIs(t, err, tc.expected)
		})
	}

	// Test non-predefined HTTP code
	err := orberrors.HTTP(404)
	require.Equal(t, 404, err.Code)
	require.Equal(t, "not found", err.Message)
}

func TestFromWithNilError(t *testing.T) {
	require.Nil(t, orberrors.From(nil))
}

func TestFromWithOrbError(t *testing.T) {
	original := orberrors.New(418, "I'm a teapot")
	result := orberrors.From(original)
	require.Same(t, original, result)
}

func TestAsWithNonOrbError(t *testing.T) {
	err := errors.New("regular error")
	orbe, ok := orberrors.As(err)
	require.False(t, ok)
	require.Nil(t, orbe)
}

func TestUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := orberrors.New(500, "outer error").Wrap(innerErr)

	unwrapped := err.Unwrap()
	require.Equal(t, innerErr, unwrapped)
}

func TestErrorIsComparison(t *testing.T) {
	// Same code and message
	err1 := orberrors.New(404, "not found")
	err2 := orberrors.New(404, "not found")
	require.ErrorIs(t, err1, err2)

	// Different code
	err3 := orberrors.New(400, "not found")
	require.NotErrorIs(t, err1, err3)

	// Different message
	err4 := orberrors.New(404, "page not found")
	require.NotErrorIs(t, err1, err4)

	// Not an orberrors.Error
	err5 := errors.New("regular error")
	require.NotErrorIs(t, err1, err5)
}

func TestMultiLevelWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	orbErr := orberrors.ErrBadRequest.Wrap(baseErr)
	wrappedErr := fmt.Errorf("outer: %w", orbErr)

	// Can still detect the orberrors.Error
	require.ErrorIs(t, wrappedErr, orberrors.ErrBadRequest)

	// Can extract the orberrors.Error
	extracted, ok := orberrors.As(wrappedErr)
	require.True(t, ok)
	require.Equal(t, 400, extracted.Code)

	// Can get to the base error
	require.ErrorIs(t, wrappedErr, baseErr)
}

func TestDefaultErrors(t *testing.T) {
	testCases := []struct {
		err     *orberrors.Error
		code    int
		message string
	}{
		{orberrors.ErrUnimplemented, 500, "Unimplemented"},
		{orberrors.ErrUnavailable, 503, "Unavailable"},
		{orberrors.ErrInternalServerError, 500, "internal server error"},
		{orberrors.ErrUnauthorized, 401, "unauthorized"},
		{orberrors.ErrRequestTimeout, 408, "request timeout"},
		{orberrors.ErrBadRequest, 400, "bad request"},
		{orberrors.ErrCanceled, 499, ""},
	}

	for _, tc := range testCases {
		testName := tc.message
		if testName == "" {
			testName = fmt.Sprintf("Code_%d", tc.code)
		}
		t.Run(testName, func(t *testing.T) {
			require.Equal(t, tc.code, tc.err.Code)
			require.Equal(t, tc.message, tc.err.Message)
		})
	}
}
