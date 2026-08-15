export type EventRow = {
  id: number;
  event_name: string;
  user_id: string;
  click_id: string;
  ad_platform: string;
  campaign_id: number;
  occurred_at: string;
  received_at: string;
};

export type ConversionRow = {
  id: number;
  ad_platform: string;
  campaign_id: number;
  click_id: string;
  status: string;
  attempt_count: number;
  last_error: string;
  updated_at: string;
};

export type Campaign = {
  id: number;
  name: string;
  ad_platform: string;
  external_campaign_id: string;
  attribution_window_days: number;
  is_active: boolean;
};

export type ExperimentSummary = {
  id: number;
  key: string;
  name: string;
  status: string;
  traffic_allocation: number;
};

export type ExperimentVariant = {
  key: string;
  name: string;
  weight: number;
};

export type ExperimentDetail = ExperimentSummary & {
  variants: ExperimentVariant[];
};

export type VariantStats = {
  variant_key: string;
  assigned: number;
  converted: number;
  conversion_rate: number;
  confidence_interval: [number, number];
  p_value_vs_control?: number;
};

export type SRMCheck = {
  chi_square: number;
  p_value: number;
  flagged: boolean;
};

export type ExperimentStatsResponse = {
  variants: VariantStats[];
  srm_check?: SRMCheck;
};

export type QueueStatus = {
  pending: number;
  active: number;
  scheduled: number;
  retry: number;
  archived: number;
  completed: number;
  processed: number;
  failed: number;
};
