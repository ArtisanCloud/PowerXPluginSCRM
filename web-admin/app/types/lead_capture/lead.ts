export interface Lead {
  lead_uuid: string;
  tenant_uuid: string;
  display_name?: string;
  phone?: string;
  email?: string;
  status: string;
  owner_user_uuid?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  created_at?: string;
  updated_at?: string;
}

export interface LeadCreatePayload {
  display_name?: string;
  phone?: string;
  email?: string;
  source_channel?: string;
  source_app_type?: string;
  source_account_uuid?: string;
  owner_user_uuid?: string;
}

export interface LeadAssignment {
  assignment_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  owner_user_uuid: string;
  reason?: string;
  created_at: string;
}

export interface LeadStatusHistory {
  history_uuid: string;
  tenant_uuid: string;
  lead_uuid: string;
  from_status: string;
  to_status: string;
  changed_at: string;
}
