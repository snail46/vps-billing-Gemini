import { apiFetch, setAuthToken, setCSRFToken } from "./client";
import { ApiResponse } from "../types/api";

export interface UserDTO {
  id: string;
  email: string;
  status: string;
  locale: string;
  timezone: string;
  email_verified_at?: string;
  last_login_at?: string;
  created_at: string;
}

export interface AdminDTO {
  id: string;
  email: string;
  display_name?: string;
  status: string;
  two_factor_enabled: boolean;
  last_login_at?: string;
  created_at: string;
}

export interface AuthLoginResponse {
  user?: UserDTO;
  admin?: AdminDTO;
  roles?: string[];
  permissions?: string[];
  token?: string;
  csrf_token: string;
}

export interface UserMeResponse {
  user: UserDTO;
}

export interface AdminMeResponse {
  admin: AdminDTO;
  roles: string[];
  permissions: string[];
}

export interface AuditEventDTO {
  id: string;
  actor_type: string;
  actor_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  ip_address?: string;
  user_agent?: string;
  request_id?: string;
  trace_id?: string;
  created_at: string;
}

export interface AuditListResponse {
  items: AuditEventDTO[];
  limit: number;
  offset: number;
}

export const authApi = {
  // User Auth
  register: (data: { email: string; password: string; locale?: string; timezone?: string }): Promise<ApiResponse<{ user: UserDTO }>> =>
    apiFetch("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  loginUser: async (data: { email: string; password: string }): Promise<ApiResponse<AuthLoginResponse>> => {
    const res = await apiFetch<AuthLoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
    if (res.success && res.data) {
      if (res.data.token) setAuthToken(res.data.token);
      if (res.data.csrf_token) setCSRFToken(res.data.csrf_token, "user");
    }
    return res;
  },

  logoutUser: async (): Promise<ApiResponse<{ logged_out: boolean }>> => {
    try {
      return await apiFetch("/api/v1/auth/logout", {
        method: "POST",
      });
    } finally {
      setAuthToken(null);
      setCSRFToken(null, "user");
    }
  },

  getMeUser: async (): Promise<ApiResponse<UserMeResponse>> => {
    const res = await apiFetch<UserMeResponse>("/api/v1/auth/me");
    if (res.success && res.data) {
      const data = res.data as any;
      if (data.token) setAuthToken(data.token);
      if (data.csrf_token) setCSRFToken(data.csrf_token, "user");
    }
    return res;
  },

  // Admin Auth
  loginAdmin: async (data: { email: string; password: string }): Promise<ApiResponse<AuthLoginResponse>> => {
    const res = await apiFetch<AuthLoginResponse>("/api/v1/admin/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
    if (res.success && res.data) {
      if (res.data.token) setAuthToken(res.data.token);
      if (res.data.csrf_token) setCSRFToken(res.data.csrf_token, "admin");
    }
    return res;
  },

  logoutAdmin: async (): Promise<ApiResponse<{ logged_out: boolean }>> => {
    try {
      return await apiFetch("/api/v1/admin/auth/logout", {
        method: "POST",
      });
    } finally {
      setAuthToken(null);
      setCSRFToken(null, "admin");
    }
  },

  getMeAdmin: async (): Promise<ApiResponse<AdminMeResponse>> => {
    const res = await apiFetch<AdminMeResponse>("/api/v1/admin/auth/me");
    if (res.success && res.data) {
      const data = res.data as any;
      if (data.token) setAuthToken(data.token);
      if (data.csrf_token) setCSRFToken(data.csrf_token, "admin");
    }
    return res;
  },

  // Admin Audit
  listAudit: (limit = 20, offset = 0): Promise<ApiResponse<AuditListResponse>> =>
    apiFetch(`/api/v1/admin/audit?limit=${limit}&offset=${offset}`),
};
