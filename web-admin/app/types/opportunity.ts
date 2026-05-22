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
  probability?: number
  owner_user_uuid: string
  source_channel?: string
  source_app_type?: string
  source_account_uuid?: string
  external_userid?: string
  expected_close_at?: string
  won_at?: string
  lost_at?: string
  lost_reason?: string
  risk_flags?: string[] | string
  created_at?: string
  updated_at?: string
}

export interface CreateOpportunityRequest {
  lead_uuid: string
  title: string
  owner_user_uuid: string
  amount?: number
  currency?: string
  expected_close_at?: string
}

export interface UpdateOpportunityRequest {
  title?: string
  owner_user_uuid?: string
  amount?: number
  currency?: string
  probability?: number
  expected_close_at?: string
}

export interface OpportunityLineItem {
  item_uuid: string
  opportunity_uuid: string
  name: string
  quantity: number
  unit_price: number
  currency: string
}

export interface OpportunityTask {
  task_uuid: string
  opportunity_uuid: string
  title: string
  due_at?: string
  status: 'open' | 'done'
}

export interface OpportunityActivity {
  activity_uuid: string
  activity_type: 'create' | 'stage_change' | 'close' | 'reopen' | 'note' | 'risk_flag' | 'line_item' | 'task'
  from_stage?: OpportunityStage
  to_stage?: OpportunityStage
  payload?: Record<string, unknown>
  operator_user_uuid: string
  created_at: string
}
