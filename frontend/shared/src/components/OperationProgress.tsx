import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Operation, OperationStep } from "../types/operation";
import { OPERATION_STATUS_META } from "../types/status";
import { operationApi } from "../api/operation";
import { StatusBadge } from "./StatusBadge";

export interface OperationProgressProps {
  operationId: string;
  onFinished?: (op: Operation) => void;
  className?: string;
}

export const OperationProgress: React.FC<OperationProgressProps> = ({
  operationId,
  onFinished,
  className = "",
}) => {
  const { t } = useTranslation();
  const [operation, setOperation] = useState<Operation | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!operationId) return;

    // Load initial
    operationApi
      .getOperation(operationId)
      .then((res) => {
        setOperation(res.operation);
        if (
          res.operation.status === "succeeded" ||
          res.operation.status === "failed" ||
          res.operation.status === "cancelled"
        ) {
          if (onFinished) onFinished(res.operation);
        }
      })
      .catch((err) => setError(err.message));

    // Subscribe live SSE
    const unsubscribe = operationApi.subscribe(
      operationId,
      (updatedOp) => {
        setOperation(updatedOp);
        if (
          updatedOp.status === "succeeded" ||
          updatedOp.status === "failed" ||
          updatedOp.status === "cancelled"
        ) {
          if (onFinished) onFinished(updatedOp);
        }
      },
      () => {
        // SSE error handled by internal fallback
      }
    );

    return () => {
      unsubscribe();
    };
  }, [operationId, onFinished]);

  if (error) {
    return (
      <div className={`p-4 rounded-lg bg-red-50 text-red-800 border border-red-200 ${className}`}>
        <p className="font-medium">{t("errors.operation_load_failed", "Failed to load operation status")}</p>
        <p className="text-sm mt-1">{error}</p>
      </div>
    );
  }

  if (!operation) {
    return (
      <div className={`p-6 rounded-lg bg-white border border-gray-200 shadow-sm animate-pulse ${className}`}>
        <div className="h-4 bg-gray-200 rounded w-1/3 mb-4"></div>
        <div className="h-2 bg-gray-200 rounded w-full mb-6"></div>
        <div className="space-y-3">
          <div className="h-3 bg-gray-200 rounded w-1/2"></div>
          <div className="h-3 bg-gray-200 rounded w-2/3"></div>
        </div>
      </div>
    );
  }

  const meta = OPERATION_STATUS_META[operation.status] || {
    severity: "neutral",
    i18nKey: "operation.status.queued",
  };

  return (
    <div className={`p-6 rounded-lg bg-white border border-gray-200 shadow-sm ${className}`}>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">
            {t(`operations.types.${operation.type}`, operation.type)}
          </h3>
          <p className="text-sm text-gray-500 mt-0.5">
            {operation.message_key
              ? t(operation.message_key, operation.phase || operation.status)
              : operation.phase || operation.status}
          </p>
        </div>
        <StatusBadge severity={meta.severity} label={t(meta.i18nKey, operation.status)} />
      </div>

      {/* Progress Bar */}
      <div className="mb-6">
        <div className="flex justify-between text-xs text-gray-500 mb-1">
          <span>{t("common.progress", "Progress")}</span>
          <span className="font-semibold">{operation.progress}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2.5 overflow-hidden">
          <div
            className={`h-2.5 rounded-full transition-all duration-300 ${
              operation.status === "failed"
                ? "bg-red-500"
                : operation.status === "succeeded"
                ? "bg-emerald-500"
                : "bg-indigo-600"
            }`}
            style={{ width: `${Math.min(100, Math.max(0, operation.progress))}%` }}
          ></div>
        </div>
      </div>

      {/* Error Banner if failed */}
      {operation.status === "failed" && (
        <div className="mb-4 p-3 rounded bg-red-50 border border-red-200 text-red-700 text-sm">
          <p className="font-semibold">{operation.error_code || t("errors.unknown", "Operation Failed")}</p>
          {operation.error_message && <p className="mt-0.5">{operation.error_message}</p>}
        </div>
      )}

      {/* Step Breakdown */}
      {operation.steps && operation.steps.length > 0 && (
        <div className="border-t border-gray-100 pt-4 mt-4">
          <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">
            {t("operations.steps_title", "Workflow Steps")}
          </h4>
          <ol className="space-y-3">
            {operation.steps.map((step: OperationStep) => {
              let stepIcon = (
                <div className="w-5 h-5 rounded-full border-2 border-gray-300 flex items-center justify-center text-xs text-gray-400">
                  {step.step_order}
                </div>
              );

              if (step.status === "running") {
                stepIcon = (
                  <div className="w-5 h-5 rounded-full border-2 border-indigo-600 border-t-transparent animate-spin"></div>
                );
              } else if (step.status === "succeeded") {
                stepIcon = (
                  <div className="w-5 h-5 rounded-full bg-emerald-500 text-white flex items-center justify-center text-xs">
                    ✓
                  </div>
                );
              } else if (step.status === "failed") {
                stepIcon = (
                  <div className="w-5 h-5 rounded-full bg-red-500 text-white flex items-center justify-center text-xs">
                    ✗
                  </div>
                );
              }

              return (
                <li key={step.id} className="flex items-start space-x-3 text-sm">
                  <div className="flex-shrink-0 mt-0.5">{stepIcon}</div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <span
                        className={`font-medium ${
                          step.status === "running"
                            ? "text-indigo-600"
                            : step.status === "succeeded"
                            ? "text-gray-900"
                            : step.status === "failed"
                            ? "text-red-600"
                            : "text-gray-400"
                        }`}
                      >
                        {t(`operations.step.${step.step_key}`, step.step_key)}
                      </span>
                      {step.status === "running" && (
                        <span className="text-xs text-indigo-500 font-semibold">{step.progress}%</span>
                      )}
                    </div>
                    {step.error_message && (
                      <p className="text-xs text-red-500 mt-0.5">{step.error_message}</p>
                    )}
                  </div>
                </li>
              );
            })}
          </ol>
        </div>
      )}
    </div>
  );
};
