package handlers

import (
	"errors"
	"strconv"
	"time"
)

func processTimeRange(start string, end string) (startTime time.Time, endTime time.Time, err error) { //nolint:nonamedreturns
	if startInt, err := strconv.ParseInt(start, 10, 64); err == nil {
		startTime = time.UnixMilli(startInt).UTC()
	}

	if endInt, err := strconv.ParseInt(end, 10, 64); err == nil {
		endTime = time.UnixMilli(endInt).UTC()
	}

	if endTime.Before(startTime) {
		return time.Time{}, time.Time{}, errors.New("end time must be greater than start time")
	}
	if endTime.Sub(startTime) > 4*time.Hour {
		return time.Time{}, time.Time{}, errors.New("time range must be 4 hours or less")
	}

	if endTime.Sub(startTime) < time.Hour {
		startTime = startTime.Truncate(time.Hour)
		endTime = startTime.Add(time.Hour - time.Nanosecond)
	}
	return startTime, endTime, nil
}
