import { apiFetch } from "./client";
import { Operation, OperationDetailResponse } from "../types/operation";

export const operationApi = {
  async getOperation(id: string): Promise<OperationDetailResponse> {
    const res = await apiFetch<OperationDetailResponse>(`/api/v1/operations/${id}`);
    if (!res.success) throw new Error(res.error?.message_key || "failed to get operation");
    return res.data;
  },

  /**
   * Subscribe to real-time operation updates via SSE, falling back to polling if SSE is disconnected.
   * Returns a cleanup function to unsubscribe.
   */
  subscribe(
    id: string,
    onUpdate: (op: Operation) => void,
    onError?: (err: Error) => void
  ): () => void {
    let closed = false;
    let pollInterval: ReturnType<typeof setInterval> | null = null;
    let eventSource: EventSource | null = null;

    const startPolling = () => {
      if (closed || pollInterval) return;
      pollInterval = setInterval(async () => {
        try {
          const res = await operationApi.getOperation(id);
          if (closed) return;
          onUpdate(res.operation);
          if (
            res.operation.status === "succeeded" ||
            res.operation.status === "failed" ||
            res.operation.status === "cancelled"
          ) {
            stopAll();
          }
        } catch (err: any) {
          if (onError) onError(err);
        }
      }, 2000);
    };

    const stopAll = () => {
      closed = true;
      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    };

    if (typeof EventSource !== "undefined") {
      const sseUrl = `/api/v1/operations/${id}/events`;
      eventSource = new EventSource(sseUrl, { withCredentials: true });

      eventSource.addEventListener("operation", (evt) => {
        try {
          const op: Operation = JSON.parse(evt.data);
          onUpdate(op);
          if (
            op.status === "succeeded" ||
            op.status === "failed" ||
            op.status === "cancelled"
          ) {
            stopAll();
          }
        } catch {
          // ignore parse error
        }
      });

      eventSource.onerror = () => {
        // SSE connection failure, fallback to polling
        if (eventSource) {
          eventSource.close();
          eventSource = null;
        }
        startPolling();
      };
    } else {
      startPolling();
    }

    return stopAll;
  },
};
