package handlers

import (
	"math"
	"strings"
	"time"

	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
)

type apiResponse []map[string]string

type ListedObject struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"` //nolint:tagliatelle
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
	ReconcileID         string `json:"reconcileID,omitempty"` //nolint:tagliatelle
	HubName             string `json:"hubName,omitempty"`
	Event               string `json:"event,omitempty"`
	Status              string `json:"status,omitempty"`
	Expression          string `json:"expression,omitempty"`
	MetricName          string `json:"metricName,omitempty"`
	Value               string `json:"value,omitempty"`
	RelevantLabelValues string `json:"relevantLabelValues,omitempty"`
}

func newLogRecord(logRecord *logspb.LogRecord, resourceLog *logspb.ResourceLogs) LogRecord {
	logRecordAttrs := logRecord.GetAttributes()
	resourceAttrs := resourceLog.GetResource().GetAttributes()

	get := func(key string) string {
		return getAttribute(logRecordAttrs, key).String()
	}
	getRes := func(key string) string {
		return getAttribute(resourceAttrs, key).String()
	}

	return LogRecord{
		Timestamp:           time.Unix(0, int64(min(logRecord.GetTimeUnixNano(), math.MaxInt64))).UTC().Format(time.RFC3339), //nolint:gosec // G115: bounded by min()
		Severity:            normalizeSeverity(logRecord.GetSeverityText()),
		SeverityNumber:      logRecord.GetSeverityNumber().String(),
		Body:                logRecord.GetBody().String(),
		Reason:              get("k8s.event.reason"),
		EventName:           get("k8s.event.name"),
		Pod:                 getRes("k8s.object.name"),
		ServiceName:         getRes("service.name"),
		Controller:          get("controller"),
		ControllerGroup:     get("controllerGroup"),
		ControllerKind:      get("controllerKind"),
		MdaiHub:             get("MdaiHub"),
		Namespace:           get("namespace"),
		Name:                get("name"),
		ReconcileID:         get("reconcileID"),
		HubName:             get("hub_name"),
		Event:               get("event"),
		Status:              get("status"),
		Expression:          get("expression"),
		MetricName:          get("metricName"),
		Value:               get("value"),
		RelevantLabelValues: get("relevantLabelValues"),
		Count:               1,
	}
}

func (lr LogRecord) key() string {
	return strings.Join([]string{
		lr.Timestamp, // or whatever format you need
		lr.Severity,
		lr.Reason,
		lr.EventName,
		lr.Pod,
		lr.ServiceName,
		lr.Body,
	}, "|")
}
