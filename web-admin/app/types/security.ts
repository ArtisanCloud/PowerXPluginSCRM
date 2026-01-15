export interface ConsentTokenResponse {
  id: string;
  tenantUuid: string;
  token: string;
  scope: string[];
  status: string;
  expiresAt?: string;
  issuedAt: string;
  issuedBy: string;
  revokedAt?: string;
  revokedReason?: string;
}

export interface ConsentTokenListResponse {
  data: ConsentTokenResponse[];
}

export interface LifecycleEventResponse {
  id: string;
  tenantUuid: string;
  eventType: string;
  assetKey: string;
  status: string;
  occurredAt: string;
  recordedBy: string;
  payload?: any;
}

export interface LifecycleEventListResponse {
  data: LifecycleEventResponse[];
}

export interface AuditReport {
  id: string;
  baseline_id: string;
  initiated_by: string;
  status: string;
  findings?: Record<string, any>;
  artifact_path?: string;
  sarif_path?: string;
  report_hash?: string;
  checklist_version: string;
  created_at: string;
}

export interface AuditReportListResponse {
  data: AuditReport[];
}

export interface ToolGrantRevocation {
  id: string;
  tenant_uuid: string;
  toolgrant_id: string;
  revoked_at: string;
  revoked_by: string;
  reason?: string;
  ttl_expiry: string;
  created_at: string;
}

export interface ToolGrantUsageEvent {
  id: string;
  tenant_uuid: string;
  toolgrant_id: string;
  event_type: string;
  capability: string;
  agent_id: string;
  occurred_at: string;
  metadata?: Record<string, any>;
}

export interface VulnerabilityAdvisory {
  id: string;
  reference: string;
  severity: string;
  status: string;
  affected_versions: string[];
  patched_in_version?: string;
  summary: string;
  details_markdown?: string;
  published_at?: string;
  patched_at?: string;
  closed_at?: string;
  sla_deadline?: string;
  created_at: string;
}

export interface VulnerabilityAdvisoryListResponse {
  data: VulnerabilityAdvisory[];
}
