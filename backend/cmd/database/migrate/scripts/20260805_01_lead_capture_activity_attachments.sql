CREATE TABLE IF NOT EXISTS lead_capture_attachments (
  attachment_uuid uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_uuid uuid NOT NULL,
  lead_uuid uuid NOT NULL,
  activity_uuid uuid NULL,
  stage_key varchar(64) NULL,
  action_key varchar(64) NULL,
  file_name text NOT NULL,
  content_type text NULL,
  file_size bigint NOT NULL DEFAULT 0,
  storage_provider varchar(32) NOT NULL DEFAULT 'database',
  content bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_lead_capture_attachments_tenant
  ON lead_capture_attachments (tenant_uuid);

CREATE INDEX IF NOT EXISTS idx_lead_capture_attachments_lead
  ON lead_capture_attachments (lead_uuid);

CREATE INDEX IF NOT EXISTS idx_lead_capture_attachments_activity
  ON lead_capture_attachments (activity_uuid);

CREATE INDEX IF NOT EXISTS idx_lead_capture_attachments_scope
  ON lead_capture_attachments (tenant_uuid, lead_uuid, stage_key, action_key);

ALTER TABLE lead_capture_attachments ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_lead_capture_attachments ON lead_capture_attachments;
CREATE POLICY tenant_isolation_lead_capture_attachments
  ON lead_capture_attachments
  USING (tenant_uuid = current_setting('app.tenant_uuid')::uuid)
  WITH CHECK (tenant_uuid = current_setting('app.tenant_uuid')::uuid);
