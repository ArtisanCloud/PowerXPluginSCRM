export interface IntegrationEnvelopePreview {
  messageId: string;
  toolScope: string;
  tenantUuid: string;
  issuedAt: string;
}

export interface IntegrationApprovalStatus {
  id: string;
  target: string;
  status: "PENDING" | "APPROVED" | "REJECTED";
  submittedBy: string;
  submittedAt: string;
}

export interface IntegrationWebhookSubscription {
  id: string;
  tenant_uuid: string;
  event_type: string;
  target_url: string;
  retry_policy?: number[];
  status: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface IntegrationWebhookAttempt {
  id: string;
  subscription_id: string;
  status: string;
  retry_count: number;
  last_error?: string;
  next_delivery_at?: string | null;
  payload_snapshot?: Record<string, any> | null;
  created_at: string;
  updated_at: string;
}

export interface IntegrationSecret {
  id: string;
  tenant_uuid: string;
  integration_type: string;
  current_secret_ref?: string;
  pending_secret_ref?: string;
  rotation_interval_days: number;
  last_rotated_at?: string | null;
  next_rotation_due_at?: string | null;
  status: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface IntegrationSecretAuditEntry {
  action: string;
  actor: string;
  timestamp: string;
  details?: Record<string, any>;
}

export type MarketplaceListingStatus =
  | "draft"
  | "in_review"
  | "published"
  | "suspended";
export type MarketplaceChecklistStatus = "pending" | "passed" | "failed";
export type MarketplaceChecklistTrigger = "vendor" | "ci" | "auto";

export interface MarketplaceListingAsset {
  id: string;
  listing_id: string;
  asset_type: string;
  storage_uri: string;
  checksum?: string;
  is_primary: boolean;
  locale: string;
  weight: number;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface MarketplacePricingTier {
  id: string;
  plan_id: string;
  metric: string;
  range_from: number;
  range_to?: number | null;
  unit_amount: number;
  unit_name?: string;
  created_at: string;
  updated_at: string;
}

export interface MarketplacePricingPlan {
  id: string;
  listing_id: string;
  plan_code: string;
  plan_type: "free" | "one_time" | "subscription" | "usage";
  currency: string;
  amount?: number | null;
  billing_period?: string;
  trial_period_days?: number | null;
  quota_limit?: number | null;
  overage_policy?: string;
  feature_matrix?: Record<string, any>;
  is_default: boolean;
  status?: string;
  created_at: string;
  updated_at: string;
  tiers?: MarketplacePricingTier[];
}

export type MarketplaceLicenseStatus =
  | "trial"
  | "active"
  | "expired"
  | "revoked"
  | "suspended";

export interface MarketplaceLicense {
  id: string;
  listing_id: string;
  plan_id: string;
  status: MarketplaceLicenseStatus;
  expires_at: string;
  token: string;
  offline_until?: string | null;
  renewal_token?: string;
  settlement_currency?: string;
  exchange_rate?: number | null;
}

export interface MarketplaceListing {
  id: string;
  plugin_id: string;
  vendor_id: string;
  status: MarketplaceListingStatus;
  title: string;
  slug: string;
  summary?: string;
  description?: string;
  cover_asset_id?: string | null;
  hero_video_asset_id?: string | null;
  categories?: string[];
  tags?: string[];
  locale: string;
  version?: string;
  ready_checklist_score: number;
  recommended_weight: number;
  published_at?: string | null;
  reviewed_at?: string | null;
  reviewer_id?: string | null;
  audit_notes?: string;
  branding_theme?: Record<string, any>;
  created_at: string;
  updated_at: string;
  assets?: MarketplaceListingAsset[];
  pricing_plans?: MarketplacePricingPlan[];
}

export interface MarketplaceChecklistItem {
  id: string;
  code: string;
  description: string;
  result: MarketplaceChecklistStatus;
  evidence_uri?: string;
  notes?: string;
  auto_fix_link?: string;
}

export interface MarketplaceChecklistRun {
  id: string;
  listing_id: string;
  run_number: number;
  trigger_source: MarketplaceChecklistTrigger;
  status: MarketplaceChecklistStatus;
  summary?: string;
  started_at: string;
  completed_at?: string | null;
  ci_pipeline_id?: string;
  items?: MarketplaceChecklistItem[];
}

export interface MarketplaceUsageDataPoint {
  timestamp: string;
  metric: string;
  value: number;
  revenue: number;
  quota_remaining?: number | null;
  window: string;
  currency?: string;
}

export interface MarketplaceUsageAlert {
  code: string;
  severity: string;
  message: string;
}

export interface MarketplaceUsageDashboard {
  series: MarketplaceUsageDataPoint[];
  alerts: MarketplaceUsageAlert[];
}

export interface MarketplaceRevenueReport {
  id: string;
  tenant_uuid: string;
  vendor_id: string;
  period_start: string;
  period_end: string;
  gross_amount: number;
  vendor_share: number;
  platform_share: number;
  fees: number;
  currency: string;
  status: string;
  generated_at: string;
  export_uri?: string | null;
}
