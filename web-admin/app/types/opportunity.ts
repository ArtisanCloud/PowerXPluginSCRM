export type OpportunityStage =
  | 'open'
  | 'qualified'
  | 'proposal'
  | 'negotiation'
  | 'won'
  | 'lost'

export interface OpportunityRecord {
  opportunity_uuid: string
  lead_uuid: string
  title: string
  stage: OpportunityStage
  amount?: number
  currency?: string
  owner_user_uuid: string
  source_channel?: string
  source_app_type?: string
  source_account_uuid?: string
  won_at?: string
  lost_at?: string
  lost_reason?: string
  risk_flags?: string[]
}

export interface CreateOpportunityRequest {
  lead_uuid: string
  title: string
  owner_user_uuid: string
  amount?: number
  currency?: string
  expected_close_at?: string
}

export interface OpportunityActivity {
  activity_uuid: string
  activity_type: 'create' | 'stage_change' | 'close' | 'reopen' | 'note' | 'risk_flag'
  from_stage?: OpportunityStage
  to_stage?: OpportunityStage
  payload?: Record<string, unknown>
  operator_user_uuid: string
  created_at: string
}
