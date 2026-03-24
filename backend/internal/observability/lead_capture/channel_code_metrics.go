package lead_capture

const (
	metricChannelCodeConfigChangeTotal = "powerx_lead_capture_channel_code_config_change_total"
	metricChannelCodeEventIngestTotal  = "powerx_lead_capture_channel_code_event_ingest_total"
	metricWelcomeSyncAttemptTotal      = "powerx_lead_capture_welcome_sync_attempt_total"
)

func (m *Metrics) RecordChannelCodeConfigChange(channel, appType string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"channel":  normalize(channel),
		"app_type": normalize(appType),
	})
	ensureCounter(m.counters, metricChannelCodeConfigChangeTotal)[labels]++
}

func (m *Metrics) RecordChannelCodeEventIngest(channel, appType, result string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"channel":  normalize(channel),
		"app_type": normalize(appType),
		"result":   normalize(result),
	})
	ensureCounter(m.counters, metricChannelCodeEventIngestTotal)[labels]++
}

func (m *Metrics) RecordWelcomeSyncAttempt(channel, appType, trigger, result, errorCode string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"channel":    normalize(channel),
		"app_type":   normalize(appType),
		"trigger":    normalize(trigger),
		"result":     normalize(result),
		"error_code": normalize(errorCode),
	})
	ensureCounter(m.counters, metricWelcomeSyncAttemptTotal)[labels]++
}
