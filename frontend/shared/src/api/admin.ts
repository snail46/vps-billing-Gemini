import { apiFetch } from './client';
import { Node, Provider } from '../types/infrastructure';
import { Instance } from '../types/instance';
import { Operation } from '../types/operation';
import { UserDTO, AdminDTO } from './auth';
import { Subscription } from '../types/subscription';

export interface HealthCheckResult {
  status: string;
  checks?: Record<string, string>;
  timestamp?: string;
}

export const adminApi = {
  listNodes: async (): Promise<Node[]> => {
    const res = await apiFetch<{ nodes: Node[] }>('/api/v1/admin/nodes');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list nodes');
    return res.data?.nodes || [];
  },

  listProviders: async (): Promise<Provider[]> => {
    const res = await apiFetch<{ providers: Provider[] }>('/api/v1/admin/providers');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list providers');
    return res.data?.providers || [];
  },

  listInstances: async (): Promise<Instance[]> => {
    const res = await apiFetch<{ instances: Instance[] }>('/api/v1/admin/instances');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list instances');
    return res.data?.instances || [];
  },

  listOperations: async (limit = 50): Promise<Operation[]> => {
    const res = await apiFetch<{ operations: Operation[] }>(`/api/v1/admin/operations?limit=${limit}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list operations');
    return res.data?.operations || [];
  },

  listUsers: async (limit = 50): Promise<UserDTO[]> => {
    const res = await apiFetch<{ users: UserDTO[] }>(`/api/v1/admin/users?limit=${limit}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list users');
    return res.data?.users || [];
  },

  listAdmins: async (): Promise<AdminDTO[]> => {
    const res = await apiFetch<{ admins: AdminDTO[] }>('/api/v1/admin/admins');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list admins');
    return res.data?.admins || [];
  },

  listSubscriptions: async (): Promise<Subscription[]> => {
    const res = await apiFetch<{ subscriptions: Subscription[] }>('/api/v1/admin/subscriptions');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list subscriptions');
    return res.data?.subscriptions || [];
  },

  checkHealth: async (): Promise<HealthCheckResult> => {
    try {
      const res = await fetch('/health/ready');
      return await res.json();
    } catch {
      return { status: 'unhealthy' };
    }
  },
};
