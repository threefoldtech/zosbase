package healthcheck

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
)

func TestMain(m *testing.M) {
	os.Setenv("ZOS_RUNMODE", "dev")
	m.Run()
}

func TestCacheCheck(t *testing.T) {
	errs := cacheCheck(context.Background())
	require.Equal(t, 0, len(errs), "Cache check should not return any errors")
	_, err := os.OpenFile("/var/cache/healthcheck", os.O_RDONLY, 0644)
	require.Error(t, err, "Cache file shouldn't exist after cache check")
}

func TestNetworkCheck(t *testing.T) {
	errors := networkCheck(context.Background())
	require.Equal(t, 0, len(errors), "Network check should not return any errors")
}

func TestGetCurrentUTCTime(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("successful response from worldtimeapi", func(t *testing.T) {
		httpmock.Reset()

		testTime := time.Date(2023, 6, 15, 12, 30, 45, 0, time.UTC)
		worldTimeResponse := `{"datetime": "2023-06-15T12:30:45.000000+00:00"}`
		httpmock.RegisterResponder("GET", "https://worldtimeapi.org/api/timezone/UTC",
			httpmock.NewStringResponder(200, worldTimeResponse))

		result, err := getCurrentUTCTime(nil)
		require.NoError(t, err)
		require.WithinDuration(t, testTime, result, time.Second)
		require.Equal(t, 1, httpmock.GetTotalCallCount())
	})

	t.Run("fallback to worldclockapi when worldtimeapi fails", func(t *testing.T) {
		httpmock.Reset()

		httpmock.RegisterResponder("GET", "https://worldtimeapi.org/api/timezone/UTC",
			httpmock.NewErrorResponder(errors.New("network error")))

		worldClockResponse := `{"currentDateTime": "2023-06-15T12:30Z"}`
		httpmock.RegisterResponder("GET", "http://worldclockapi.com/api/json/utc/now",
			httpmock.NewStringResponder(200, worldClockResponse))

		result, err := getCurrentUTCTime(nil)
		require.NoError(t, err)
		expectedTime := time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC)
		require.Equal(t, expectedTime, result)
		require.Equal(t, 2, httpmock.GetTotalCallCount())
	})

	t.Run("fallback to timeapi.io when first two fail", func(t *testing.T) {
		httpmock.Reset()

		httpmock.RegisterResponder("GET", "https://worldtimeapi.org/api/timezone/UTC",
			httpmock.NewErrorResponder(errors.New("network error")))

		httpmock.RegisterResponder("GET", "http://worldclockapi.com/api/json/utc/now",
			httpmock.NewStringResponder(500, "Internal Server Error"))

		timeAPIResponse := `{"dateTime": "2023-06-15T12:30:45.123456"}`
		httpmock.RegisterResponder("GET", "https://timeapi.io/api/Time/current/zone?timeZone=UTC",
			httpmock.NewStringResponder(200, timeAPIResponse))

		result, err := getCurrentUTCTime(nil)
		require.NoError(t, err)
		expectedTime := time.Date(2023, 6, 15, 12, 30, 45, 123456000, time.UTC)
		require.Equal(t, expectedTime, result)
		require.Equal(t, 3, httpmock.GetTotalCallCount())
	})

	t.Run("error when all servers fail", func(t *testing.T) {
		httpmock.Reset()

		httpmock.RegisterResponder("GET", "https://worldtimeapi.org/api/timezone/UTC",
			httpmock.NewErrorResponder(errors.New("network error")))
		httpmock.RegisterResponder("GET", "http://worldclockapi.com/api/json/utc/now",
			httpmock.NewStringResponder(500, "Internal Server Error"))
		httpmock.RegisterResponder("GET", "https://timeapi.io/api/Time/current/zone?timeZone=UTC",
			httpmock.NewErrorResponder(errors.New("timeout")))

		result, err := getCurrentUTCTime(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get time from all servers")
		require.True(t, result.IsZero())
		require.Equal(t, 3, httpmock.GetTotalCallCount())
	})

	t.Run("invalid JSON response handling", func(t *testing.T) {
		httpmock.Reset()

		httpmock.RegisterResponder("GET", "https://worldtimeapi.org/api/timezone/UTC",
			httpmock.NewStringResponder(200, "invalid json"))

		worldClockResponse := `{"currentDateTime": "2023-06-15T12:30Z"}`
		httpmock.RegisterResponder("GET", "http://worldclockapi.com/api/json/utc/now",
			httpmock.NewStringResponder(200, worldClockResponse))

		var zcl zbus.Client
		result, err := getCurrentUTCTime(zcl)
		require.NoError(t, err)
		expectedTime := time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC)
		require.Equal(t, expectedTime, result)
		require.Equal(t, 2, httpmock.GetTotalCallCount())
	})
}
