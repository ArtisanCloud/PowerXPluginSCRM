-- Opportunity partial unique index: one active primary opportunity per lead in one tenant.
CREATE UNIQUE INDEX IF NOT EXISTS uq_opp_active_per_lead
ON opportunity_records (tenant_uuid, lead_uuid)
WHERE stage IN ('open', 'qualified', 'proposal', 'negotiation');

CREATE INDEX IF NOT EXISTS idx_opp_tenant_owner_stage
ON opportunity_records (tenant_uuid, owner_user_uuid, stage);

CREATE INDEX IF NOT EXISTS idx_opp_act_tenant_opp_created
ON opportunity_activities (tenant_uuid, opportunity_uuid, created_at DESC);
