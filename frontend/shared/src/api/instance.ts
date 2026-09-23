import { apiFetch } from './client';
import { Instance, InstanceActionResponse } from '../types/instance';

export const instanceApi = {
  list: async (): Promise<Instance[]> => {
    const res = await apiFetch<{ instances: Instance[] }>('/api/v1/instances');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list instances');
    return res.data?.instances || [];
  },

  get: async (id: string): Promise<Instance> => {
    const res = await apiFetch<{ instance: Instance }>(`/api/v1/instances/${id}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to get instance');
    return res.data.instance;
  },

  start: async (id: string): Promise<InstanceActionResponse> => {
    const res = await apiFetch<InstanceActionResponse>(`/api/v1/instances/${id}/start`, {
      method: 'POST',
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to start instance');
    return res.data;
  },

  stop: async (id: string): Promise<InstanceActionResponse> => {
    const res = await apiFetch<InstanceActionResponse>(`/api/v1/instances/${id}/stop`, {
      method: 'POST',
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to stop instance');
    return res.data;
  },

  restart: async (id: string): Promise<InstanceActionResponse> => {
    const res = await apiFetch<InstanceActionResponse>(`/api/v1/instances/${id}/restart`, {
      method: 'POST',
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to restart instance');
    return res.data;
  },

  reinstall: async (id: string, image: string): Promise<InstanceActionResponse> => {
    const res = await apiFetch<InstanceActionResponse>(`/api/v1/instances/${id}/reinstall`, {
      method: 'POST',
      body: JSON.stringify({ image }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to reinstall instance');
    return res.data;
  },

  renew: async (id: string): Promise<{ instance: Instance; subscription: any }> => {
    const res = await apiFetch<{ instance: Instance; subscription: any }>(`/api/v1/instances/${id}/renew`, {
      method: 'POST',
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to renew instance');
    return res.data;
  },
};
