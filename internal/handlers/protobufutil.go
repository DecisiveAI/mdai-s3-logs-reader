package handlers

import (
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

func getAttribute(attributes []*commonpb.KeyValue, key string) *commonpb.AnyValue {
	for _, attr := range attributes {
		if attr.GetKey() == key {
			return attr.GetValue()
		}
	}
	return nil
}
