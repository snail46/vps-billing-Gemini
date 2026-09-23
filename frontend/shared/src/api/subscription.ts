import { apiFetch } from "./client";
import { Subscription, CancelSubscriptionRequest } from "../types/subscription";

export const subscriptionApi = {
  async listUserSubscriptions(): Promise<{ subscriptions: Subscription[] }> {
    const res = await apiFetch<{ subscriptions: Subscription[] }>("/api/v1/subscriptions");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list subscriptions");
    return res.data;
  },

  async getSubscription(id: string): Promise<{ subscription: Subscription }> {
    const res = await apiFetch<{ subscription: Subscription }>(`/api/v1/subscriptions/${id}`);
    if (!res.success) throw new Error(res.error?.message_key || "failed to get subscription");
    return res.data;
  },

  async cancelSubscription(id: string, req: CancelSubscriptionRequest = {}): Promise<{ subscription: Subscription }> {
    const res = await apiFetch<{ subscription: Subscription }>(`/api/v1/subscriptions/${id}/cancel`, {
      method: "POST",
      body: JSON.stringify(req),
    });
    if (!res.success) throw new Error(res.error?.message_key || "failed to cancel subscription");
    return res.data;
  },

  async adminListSubscriptions(): Promise<{ subscriptions: Subscription[] }> {
    const res = await apiFetch<{ subscriptions: Subscription[] }>("/api/v1/admin/subscriptions");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list subscriptions for admin");
    return res.data;
  },
};
