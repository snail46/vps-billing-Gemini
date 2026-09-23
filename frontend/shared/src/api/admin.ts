import { apiFetch } from './client';
import { Node, Provider } from '../types/infrastructure';
import { Instance } from '../types/instance';
import { Operation } from '../types/operation';
import { UserDTO, AdminDTO } from './auth';
import { Subscription } from '../types/subscription';
import { Product, Plan } from '../types/commerce';

export interface HealthCheckResult {
  status: string;
  checks?: Record<string, string>;
  timestamp?: string;
}

export interface CreateProviderInput {
  name: string;
  provider_type: string;
  status?: string;
}

export interface CreateNodeInput {
  name: string;
  provider_id?: string;
  region: string;
  cpu_total: number;
  memory_total_mb: number;
  disk_total_gb: number;
}

export interface CreateProductInput {
  slug: string;
  name_i18n: Record<string, string>;
  description_i18n?: Record<string, string>;
  status?: string;
}

export interface CreatePlanInput {
  product_id: string;
  slug: string;
  name_i18n: Record<string, string>;
  status?: string;
  cpu_cores: number;
  memory_mb: number;
  disk_gb: number;
  traffic_gb?: number;
  bandwidth_mbps?: number;
  price_minor: number;
  currency: string;
  billing_cycle?: string;
}

export const adminApi = {
  listNodes: async (): Promise<Node[]> => {
    const res = await apiFetch<{ nodes: Node[] }>('/api/v1/admin/nodes');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list nodes');
    return res.data?.nodes || [];
  },

  createNode: async (input: CreateNodeInput): Promise<Node> => {
    const res = await apiFetch<{ node: Node }>('/api/v1/admin/nodes', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to create node');
    return res.data!.node;
  },

  listProviders: async (): Promise<Provider[]> => {
    const res = await apiFetch<{ providers: Provider[] }>('/api/v1/admin/providers');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list providers');
    return res.data?.providers || [];
  },

  createProvider: async (input: CreateProviderInput): Promise<Provider> => {
    const res = await apiFetch<{ provider: Provider }>('/api/v1/admin/providers', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to create provider');
    return res.data!.provider;
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

  getOperation: async (id: string): Promise<Operation> => {
    const res = await apiFetch<{ operation: Operation }>(`/api/v1/admin/operations/${id}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to get operation');
    return res.data!.operation;
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

  createProduct: async (input: CreateProductInput): Promise<Product> => {
    const res = await apiFetch<{ product: Product }>('/api/v1/admin/products', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to create product');
    return res.data!.product;
  },

  createPlan: async (input: CreatePlanInput): Promise<Plan> => {
    const res = await apiFetch<{ plan: Plan }>('/api/v1/admin/plans', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to create plan');
    return res.data!.plan;
  },

  setup2FA: async (): Promise<{ secret: string; otpauth_url: string }> => {
    const res = await apiFetch<{ secret: string; otpauth_url: string }>('/api/v1/admin/auth/2fa/setup');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to setup 2fa');
    return res.data!;
  },

  enable2FA: async (secret: string, passcode: string): Promise<boolean> => {
    const res = await apiFetch<{ enabled: boolean }>('/api/v1/admin/auth/2fa/enable', {
      method: 'POST',
      body: JSON.stringify({ secret, passcode }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to enable 2fa');
    return res.data?.enabled ?? true;
  },

  disable2FA: async (passcode: string): Promise<boolean> => {
    const res = await apiFetch<{ disabled: boolean }>('/api/v1/admin/auth/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ passcode }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to disable 2fa');
    return res.data?.disabled ?? true;
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
