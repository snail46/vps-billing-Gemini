import { ApiResponse } from "../types/api";

export function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp("(^| )" + name + "=([^;]+)"));
  return match ? decodeURIComponent(match[2]) : null;
}

export function setAuthToken(token: string | null) {
  if (typeof window === "undefined") return;
  if (token) {
    localStorage.setItem("vps_auth_token", token);
  } else {
    localStorage.removeItem("vps_auth_token");
  }
}

export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("vps_auth_token");
}

export function setCSRFToken(token: string | null, scope?: "admin" | "user") {
  if (typeof window === "undefined") return;
  if (token) {
    if (scope) {
      localStorage.setItem(`vps_csrf_token_${scope}`, token);
    }
    localStorage.setItem("vps_csrf_token", token);
  } else {
    if (scope) {
      localStorage.removeItem(`vps_csrf_token_${scope}`);
    }
    localStorage.removeItem("vps_csrf_token");
  }
}

export function getStoredCSRFToken(scope?: "admin" | "user"): string | null {
  if (typeof window === "undefined") return null;
  if (scope) {
    const scoped = localStorage.getItem(`vps_csrf_token_${scope}`);
    if (scoped) return scoped;
  }
  return localStorage.getItem("vps_csrf_token");
}

export async function apiFetch<T>(
  url: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const headers = new Headers(options.headers || {});

  if (!headers.has("Content-Type") && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }

  // Auto-attach Bearer token if present
  if (!headers.has("Authorization")) {
    const token = getAuthToken();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  // Auto-attach CSRF Token for mutating methods with actor scoping to prevent localhost port collision
  const method = (options.method || "GET").toUpperCase();
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
    const isAdmin = url.includes("/admin");
    let csrfToken: string | null = null;
    if (isAdmin) {
      csrfToken =
        getCookie("vps_csrf_token_admin") ||
        getStoredCSRFToken("admin") ||
        getCookie("vps_csrf_token") ||
        getStoredCSRFToken();
    } else {
      csrfToken =
        getCookie("vps_csrf_token_user") ||
        getStoredCSRFToken("user") ||
        getCookie("vps_csrf_token") ||
        getStoredCSRFToken();
    }

    if (csrfToken && !headers.has("X-CSRF-Token")) {
      headers.set("X-CSRF-Token", csrfToken);
    }
  }

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: "include", // Send HttpOnly session cookies
  });

  try {
    const json = await response.json();
    return json as ApiResponse<T>;
  } catch {
    return {
      success: false,
      error: {
        code: "PARSE_ERROR",
        message_key: "errors.internal_error",
        details: "Invalid JSON response from server",
      },
    } as ApiResponse<T>;
  }
}

