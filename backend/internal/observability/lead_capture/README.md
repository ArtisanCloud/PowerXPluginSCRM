# Lead Capture Observability Checklist

## Scope (Phase 6 / T049)

This module exposes cross-cutting metrics for US1~US3 verification:

- `powerx_lead_capture_sync_task_total{provider,status}`
  - Check task provider dimension (`framework|local_fallback`) and status flow.
- `powerx_lead_capture_conversation_event_total{provider,result}`
  - Check idempotent ingestion counters (`ingest_created`, `ingest_duplicate`, `ingest_failed`, etc.).
- `powerx_lead_capture_conversation_duplicate_rate{provider}`
  - Derived duplicate drop rate = `ingest_duplicate / (ingest_created + ingest_duplicate)`.
- `powerx_lead_capture_conversation_latency_ms{provider}`
  - Latest webhook ingestion latency (ms).
- `powerx_lead_capture_conversation_latency_p95_ms{provider}`
  - Rolling p95 latency (window: 200 samples).

## Runtime Validation

1. Trigger US1 sync in both provider modes (`framework` and `local_fallback`).
2. Replay the same webhook payload (`external_event_id` unchanged) twice.
3. Check exported metrics:
   - Provider labels are present.
   - Duplicate rate rises above 0 after replay.
   - p95 latency is emitted and within expected range.
