export interface ApiSuccessResponse<T> {
  success: true;
  data: T;
  request_id: string;
}

export interface ApiErrorDetail {
  code: string;
  message_key: string;
  details?: unknown;
}

export interface ApiErrorResponse {
  success: false;
  error: ApiErrorDetail;
  request_id: string;
}

export type ApiResponse<T> = ApiSuccessResponse<T> | ApiErrorResponse;

export interface HealthLiveResponse {
  status: string;
}

export interface HealthReadyResponse {
  status: "ready" | "unready";
  timestamp: string;
  checks: Record<string, string>;
}
