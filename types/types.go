package types

import (
	"time"
)

type ListedObject struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
}

type LogRecord struct {
	Timestamp      string `json:"timestamp,omitempty"`
	Severity       string `json:"severity,omitempty"`
	SeverityNumber string `json:"severityNumber,omitempty"`
	Body           string `json:"body,omitempty"`
	Reason         string `json:"reason,omitempty"`
	EventName      string `json:"eventName,omitempty"`
	Pod            string `json:"pod,omitempty"`
	ServiceName    string `json:"serviceName,omitempty"`
	Count          int    `json:"count"`
}
