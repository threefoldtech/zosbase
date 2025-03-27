package zos

import (
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestValidBackend(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("1.1.1.1:10")
		err := backend.Valid(true)
		require.NoError(err)

		backend = Backend("1.1.1.1:10")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("1.1.1.1")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("1.1.1.1:port")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("ip:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("1.1.1.1:1000000")
		err = backend.Valid(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1:10")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://1.1.1.1:port")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("http://ip:10")
		err = backend.Valid(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://ip")
		err = backend.Valid(false)
		require.Error(err)
	})
}

func TestAsAddress(t *testing.T) {

	cases := []struct {
		Backend Backend
		Address string
		Err     bool
	}{
		{
			Backend: Backend("http://10.20.30.40"),
			Address: "10.20.30.40:80",
		},
		{
			Backend: Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200"),
			Address: "[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200",
		},
		{
			Backend: Backend("2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF:200"),
			Err:     true,
		},
		{
			Backend: Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200"),
			Address: "[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:200",
		},
		{
			Backend: Backend("http://10.20.30.40:500"),
			Address: "10.20.30.40:500",
		},
		{
			Backend: Backend("10.20.30.40:500"),
			Address: "10.20.30.40:500",
		},
	}

	for _, c := range cases {
		t.Run(string(c.Backend), func(t *testing.T) {
			addr, err := c.Backend.AsAddress()
			if c.Err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.Address, addr)
			}
		})
	}
}

func TestValidBackendIP6(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err := backend.Valid(true)
		require.NoError(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err = backend.Valid(false)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:1000000")
		err = backend.Valid(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		err = backend.Valid(true)
		require.Error(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		err = backend.Valid(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		err := backend.Valid(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		err = backend.Valid(true)
		require.Error(err)
	})
}

func TestValidateBackends(t *testing.T) {
	require := require.New(t)

	t.Run("empty backends", func(t *testing.T) {
		backends := []Backend{
			"",
		}
		err := ValidateBackends(backends, true)
		require.Error(err)

		err = ValidateBackends(backends, false)
		require.Error(err)
	})

	t.Run("all valid backends with tlsPassthrough=true", func(t *testing.T) {
		backends := []Backend{
			"1.1.1.1:80",
			"2.2.2.2:443",
			"[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:8080",
		}
		err := ValidateBackends(backends, true)
		require.NoError(err)
	})

	t.Run("all valid backends with tlsPassthrough=false", func(t *testing.T) {
		backends := []Backend{
			"http://1.1.1.1",
			"http://2.2.2.2:443",
			"http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]",
		}
		err := ValidateBackends(backends, false)
		require.NoError(err)
	})

	t.Run("mixed valid and invalid backends with tlsPassthrough=true", func(t *testing.T) {
		backends := []Backend{
			"1.1.1.1:80",
			"http://2.2.2.2:443", // invalid (should be IP:port without http://)
			"2.2.2.2",            // invalid (missing port)
			"3.3.3.3:port",       // invalid (non-numeric port)
			"127.0.0.1:8080",
			"[::1]:8080",
			"[2001:db8::1]:8080",
			"2001:db8::1:8080", // invalid (wrong IPv6 format)
		}
		err := ValidateBackends(backends, true)
		require.Error(err)
		merr, ok := err.(*multierror.Error)
		require.True(ok)
		require.Equal(4, len(merr.Errors))
	})

	t.Run("mixed valid and invalid backends with tlsPassthrough=false", func(t *testing.T) {
		backends := []Backend{
			"http://1.1.1.1",
			"1.1.1.1:80", // invalid (needs http://)
			"http://2.2.2.2:443",
			"https://3.3.3.3",  // invalid (wrong scheme)
			"http://localhost", // invalid (loopback)
			"http://127.0.0.1", // invalid (loopback)
			"http://[::1]",     // invalid (loopback)
			"http://[2001:db8::1]:8080",
		}
		err := ValidateBackends(backends, false)
		require.Error(err)
		// Check that we have the expected number of errors
		merr, ok := err.(*multierror.Error)
		require.True(ok)
		require.Equal(5, len(merr.Errors))
	})

	t.Run("scheme mismatch using https when not permitted", func(t *testing.T) {
		backends := []Backend{
			"https://1.1.1.1",
		}
		err := ValidateBackends(backends, false)
		require.Error(err)
	})

	t.Run("scheme mismatch using http when tlsPassthrough=true", func(t *testing.T) {
		backends := []Backend{
			"http://1.1.1.1:80",
		}
		err := ValidateBackends(backends, true)
		require.Error(err)
	})

	t.Run("all invalid backends", func(t *testing.T) {
		backends := []Backend{
			"invalid",
			"1.1.1.1:port",
			"http://invalid",
			"ftp://1.1.1.1",
		}

		err := ValidateBackends(backends, true)
		require.Error(err)
		merr, ok := err.(*multierror.Error)
		require.True(ok)
		require.Equal(4, len(merr.Errors))

		err = ValidateBackends(backends, false)
		require.Error(err)
		merr, ok = err.(*multierror.Error)
		require.True(ok)
		require.Equal(4, len(merr.Errors))
	})
}
