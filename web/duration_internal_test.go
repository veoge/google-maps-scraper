package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}

	return t
}

func TestDurationUsesRecordedTimings(t *testing.T) {
	t.Parallel()

	start := at("2026-09-07T10:00:00Z")
	end := at("2026-09-07T10:18:42Z")

	job := Job{
		Status: StatusOK,
		Date:   at("2026-09-07T09:00:00Z"),
		Data:   JobData{StartedAt: &start, FinishedAt: &end},
	}

	d, ok := job.Duration()
	require.True(t, ok)
	require.Equal(t, 18*time.Minute+42*time.Second, d,
		"the hour spent queued must not count as run time")
	require.Equal(t, int64(1122), job.DurationSeconds())
	require.Equal(t, start.Unix(), job.StartedUnix())
}

// The worker runs one job at a time, so a job can sit pending for a long while.
// Counting that wait as run time would be wrong, which is why StartedAt exists.
func TestQueueWaitIsExcluded(t *testing.T) {
	t.Parallel()

	created := at("2026-09-07T09:00:00Z")
	start := at("2026-09-07T11:00:00Z")
	end := at("2026-09-07T11:05:00Z")

	job := Job{
		Status: StatusOK,
		Date:   created,
		Data:   JobData{StartedAt: &start, FinishedAt: &end},
	}

	d, _ := job.Duration()
	require.Equal(t, 5*time.Minute, d)
}

// Jobs that finished before timings were recorded still report something useful.
func TestDurationFallsBackToRowTimestamps(t *testing.T) {
	t.Parallel()

	job := Job{
		Status:    StatusOK,
		Date:      at("2026-09-07T10:00:00Z"),
		UpdatedAt: at("2026-09-07T10:07:30Z"),
	}

	d, ok := job.Duration()
	require.True(t, ok)
	require.Equal(t, 7*time.Minute+30*time.Second, d)
}

func TestDurationUnknownWhenNothingRecorded(t *testing.T) {
	t.Parallel()

	job := Job{Status: StatusOK}

	_, ok := job.Duration()
	require.False(t, ok)
	require.Zero(t, job.DurationSeconds())
	require.Zero(t, job.StartedUnix())
}

// A running job reports elapsed time so far, and the table renders it as a live
// counter rather than a finished total.
func TestRunningJobReportsElapsedButNoTotal(t *testing.T) {
	t.Parallel()

	start := time.Now().UTC().Add(-90 * time.Second)
	job := Job{Status: StatusWorking, Data: JobData{StartedAt: &start}}

	d, ok := job.Duration()
	require.True(t, ok)
	require.InDelta(t, 90.0, d.Seconds(), 5)

	require.Zero(t, job.DurationSeconds(),
		"an unfinished job has no total; the browser counts it up instead")
	require.Equal(t, start.Unix(), job.StartedUnix())
}

func TestMarkStartedAndFinished(t *testing.T) {
	t.Parallel()

	job := Job{Status: StatusPending}

	job.MarkStarted()
	require.Equal(t, StatusWorking, job.Status)
	require.NotNil(t, job.Data.StartedAt)
	require.Nil(t, job.Data.FinishedAt)

	job.MarkFinished(StatusOK)
	require.Equal(t, StatusOK, job.Status)
	require.NotNil(t, job.Data.FinishedAt)

	d, ok := job.Duration()
	require.True(t, ok)
	require.GreaterOrEqual(t, d, time.Duration(0))
}

// A failed run still has to report how long it took before it gave up.
func TestFailedJobReportsDuration(t *testing.T) {
	t.Parallel()

	start := at("2026-09-07T10:00:00Z")
	end := at("2026-09-07T10:02:00Z")

	job := Job{
		Status: StatusFailed,
		Data:   JobData{StartedAt: &start, FinishedAt: &end},
	}

	require.Equal(t, int64(120), job.DurationSeconds())
}
