export type StatusSeverity = "success" | "neutral" | "info" | "warning" | "error";

export type InstanceState =
  | "running"
  | "stopped"
  | "provisioning"
  | "restarting"
  | "reinstalling"
  | "suspended"
  | "unknown"
  | "error";

export type OperationStatus =
  | "queued"
  | "running"
  | "waiting_provider"
  | "waiting_resource"
  | "verifying"
  | "retrying"
  | "succeeded"
  | "failed"
  | "cancelled";

export interface StatusMetadata {
  severity: StatusSeverity;
  i18nKey: string;
}

export const INSTANCE_STATUS_META: Record<InstanceState, StatusMetadata> = {
  running: { severity: "success", i18nKey: "instance.status.running" },
  stopped: { severity: "neutral", i18nKey: "instance.status.stopped" },
  provisioning: { severity: "info", i18nKey: "instance.status.provisioning" },
  restarting: { severity: "info", i18nKey: "instance.status.restarting" },
  reinstalling: { severity: "info", i18nKey: "instance.status.reinstalling" },
  suspended: { severity: "warning", i18nKey: "instance.status.suspended" },
  unknown: { severity: "warning", i18nKey: "instance.status.unknown" },
  error: { severity: "error", i18nKey: "instance.status.error" },
};

export const OPERATION_STATUS_META: Record<OperationStatus, StatusMetadata> = {
  queued: { severity: "info", i18nKey: "operation.status.queued" },
  running: { severity: "info", i18nKey: "operation.status.running" },
  waiting_provider: { severity: "info", i18nKey: "operation.status.waiting_provider" },
  waiting_resource: { severity: "info", i18nKey: "operation.status.waiting_resource" },
  verifying: { severity: "info", i18nKey: "operation.status.verifying" },
  retrying: { severity: "warning", i18nKey: "operation.status.retrying" },
  succeeded: { severity: "success", i18nKey: "operation.status.succeeded" },
  failed: { severity: "error", i18nKey: "operation.status.failed" },
  cancelled: { severity: "neutral", i18nKey: "operation.status.cancelled" },
};
