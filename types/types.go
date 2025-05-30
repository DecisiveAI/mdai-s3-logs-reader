package types

import (
	"time"
)

type ListedObject struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
}

type LogRecord struct {
	Timestamp           string `json:"timestamp,omitempty"`
	Severity            string `json:"severity,omitempty"`
	SeverityNumber      string `json:"severityNumber,omitempty"`
	Body                string `json:"body,omitempty"`
	Reason              string `json:"reason,omitempty"`
	EventName           string `json:"eventName,omitempty"`
	Pod                 string `json:"pod,omitempty"`
	ServiceName         string `json:"serviceName,omitempty"`
	Count               int    `json:"count"`
	Controller          string `json:"controller,omitempty"`
	ControllerGroup     string `json:"controllerGroup,omitempty"`
	ControllerKind      string `json:"controllerKind,omitempty"`
	MdaiHub             string `json:"mdaiHub,omitempty"`
	Namespace           string `json:"namespace,omitempty"`
	Name                string `json:"name,omitempty"`
	ReconcileID         string `json:"reconcileID,omitempty"`
	HubName             string `json:"hubName,omitempty"`
	Event               string `json:"event,omitempty"`
	Status              string `json:"status,omitempty"`
	Expression          string `json:"expression,omitempty"`
	MetricName          string `json:"metricName,omitempty"`
	Value               string `json:"value,omitempty"`
	RelevantLabelValues string `json:"relevantLabelValues,omitempty"`
}
