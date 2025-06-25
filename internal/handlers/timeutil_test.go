package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		startString string
		endString   string
		startTime   time.Time
		endTime     time.Time
		err         error
	}{
		{
			name:        "NoError",
			startString: "1751044132000",
			endString:   "1751052732000",
			startTime:   time.Date(2025, time.June, 27, 17, 8, 52, 0, time.UTC),
			endTime:     time.Date(2025, time.June, 27, 19, 32, 12, 0, time.UTC),
			err:         nil,
		},
		{
			name:        "StartTimeGreaterThanEndTime",
			startString: "1751044132000",
			endString:   "1751044131000",
			startTime:   time.Time{},
			endTime:     time.Time{},
			err:         errors.New("end time must be greater than start time"),
		},
		{
			name:        "TimeRangeGreaterThanFourHours",
			startString: "1731044132000",
			endString:   "1751044132000",
			startTime:   time.Time{},
			endTime:     time.Time{},
			err:         errors.New("time range must be 4 hours or less"),
		},
		{
			name:        "TimeRangeLessThanAnHour",
			startString: "1751044132000",
			endString:   "1751044133000",
			startTime:   time.Date(2025, time.June, 27, 17, 0, 0, 0, time.UTC),
			endTime:     time.Date(2025, time.June, 27, 17, 59, 59, 999999999, time.UTC),
			err:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := processTimeRange(tt.startString, tt.endString)
			assert.Equal(t, tt.startTime, start)
			assert.Equal(t, tt.endTime, end)
			require.Equal(t, err, tt.err)
		})
	}
}
