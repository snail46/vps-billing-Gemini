export type SubscriptionStatus =
  | "active"
  | "past_due"
  | "suspended"
  | "terminated"
  | "cancelled";

export interface Subscription {
  id: string;
  user_id: string;
  plan_id: string;
  status: SubscriptionStatus;
  billing_cycle: string;
  current_period_start: string;
  current_period_end: string;
  next_renewal_at?: string;
  cancel_at_period_end: boolean;
  cancelled_at?: string;
  suspended_at?: string;
  terminated_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SubscriptionListResponse {
  subscriptions: Subscription[];
}

export interface SubscriptionDetailResponse {
  subscription: Subscription;
}

export interface CancelSubscriptionRequest {
  immediate?: boolean;
}
