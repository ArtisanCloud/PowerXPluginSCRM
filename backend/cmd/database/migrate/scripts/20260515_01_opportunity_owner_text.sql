ALTER TABLE opportunity_records
  ALTER COLUMN owner_user_uuid TYPE text
  USING owner_user_uuid::text;
