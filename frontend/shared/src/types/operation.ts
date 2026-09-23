import { OperationStatus } from "./status";

export type { OperationStatus };

export type StepStatus = "pending" | "running" | "succeeded" | "failed" | "skipped";

export interface OperationStep {
  id: string;
  operation_id: string;
  step_key: string;
  step_order: number;
  status: StepStatus;
  progress: number;
  attempt: number;
  error_code?: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Operation {
  id: string;
  type: string;
  resource_type: string;
  resource_id: string;
  status: OperationStatus;
  phase?: string;
  progress: number;
  message_key?: string;
  provider_id?: string;
  provider_operation_id?: string;
  idempotency_key: string;
  retryable: boolean;
  retry_count: number;
  max_retries: number;
  error_code?: string;
  error_message?: string;
  trace_id: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
  steps?: OperationStep[];
}

export interface OperationDetailResponse {
  operation: Operation;
}
