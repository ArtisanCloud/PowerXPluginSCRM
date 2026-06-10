ALTER TABLE opportunity_line_items
  ADD COLUMN IF NOT EXISTS kind varchar(24) NOT NULL DEFAULT 'manual',
  ADD COLUMN IF NOT EXISTS total_amount numeric(18,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS storage_provider varchar(32),
  ADD COLUMN IF NOT EXISTS object_key text,
  ADD COLUMN IF NOT EXISTS file_name text,
  ADD COLUMN IF NOT EXISTS file_size bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS content_type varchar(128);

UPDATE opportunity_line_items
SET total_amount = COALESCE(quantity, 0) * COALESCE(unit_price, 0)
WHERE total_amount = 0;

CREATE INDEX IF NOT EXISTS idx_opp_line_tenant_opp_kind
ON opportunity_line_items (tenant_uuid, opportunity_uuid, kind);
