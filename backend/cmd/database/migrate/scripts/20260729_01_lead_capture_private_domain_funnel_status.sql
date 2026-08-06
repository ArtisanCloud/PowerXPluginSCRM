UPDATE lead_capture_leads
SET status = CASE status
  WHEN 'new' THEN 'captured'
  WHEN 'assigned' THEN 'routed'
  WHEN 'in_progress' THEN 'engaging'
  WHEN 'mql' THEN 'qualified_for_handoff'
  WHEN 'sql' THEN 'handoff_pending'
  WHEN 'converted' THEN 'handoff_accepted'
  WHEN 'closed' THEN 'archived'
  ELSE status
END
WHERE status IN ('new', 'assigned', 'in_progress', 'mql', 'sql', 'converted', 'closed');

UPDATE lead_capture_status_history
SET from_status = CASE from_status
  WHEN 'new' THEN 'captured'
  WHEN 'assigned' THEN 'routed'
  WHEN 'in_progress' THEN 'engaging'
  WHEN 'mql' THEN 'qualified_for_handoff'
  WHEN 'sql' THEN 'handoff_pending'
  WHEN 'converted' THEN 'handoff_accepted'
  WHEN 'closed' THEN 'archived'
  ELSE from_status
END,
to_status = CASE to_status
  WHEN 'new' THEN 'captured'
  WHEN 'assigned' THEN 'routed'
  WHEN 'in_progress' THEN 'engaging'
  WHEN 'mql' THEN 'qualified_for_handoff'
  WHEN 'sql' THEN 'handoff_pending'
  WHEN 'converted' THEN 'handoff_accepted'
  WHEN 'closed' THEN 'archived'
  ELSE to_status
END
WHERE from_status IN ('new', 'assigned', 'in_progress', 'mql', 'sql', 'converted', 'closed')
   OR to_status IN ('new', 'assigned', 'in_progress', 'mql', 'sql', 'converted', 'closed');

ALTER TABLE lead_capture_leads
ALTER COLUMN status SET DEFAULT 'captured';
