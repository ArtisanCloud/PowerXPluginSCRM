package logger

import "github.com/sirupsen/logrus"

var runtimeFieldMasker func(Fields) Fields

// RegisterRuntimeMasker sets a hook that can rewrite runtime log fields before
// they are emitted, enabling downstream privacy masking.
func RegisterRuntimeMasker(masker func(Fields) Fields) {
	runtimeFieldMasker = masker
}

// WithRuntimeFields enriches the log entry with standard runtime metadata.
func WithRuntimeFields(pluginID, tenantID, traceID, component string, extra Fields) *logrus.Entry {
	fields := Fields{
		"plugin_id": pluginID,
		"tenant_uuid": tenantID,
		"trace_id":  traceID,
		"component": component,
	}
	for k, v := range extra {
		fields[k] = v
	}
	if runtimeFieldMasker != nil {
		fields = runtimeFieldMasker(fields)
	}
	return WithFields(fields)
}
