package types

import "time"

type ListedObject struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
}

type LogRecord struct {
	Timestamp  string `json:"timestamp"`
	Severity   string `json:"severity"`
	Body       string `json:"body"`
	Reason     string `json:"reason"`
	EventName  string `json:"eventName"`
	Kind       string `json:"kind"`
	ObjectName string `json:"objectName"`
}
