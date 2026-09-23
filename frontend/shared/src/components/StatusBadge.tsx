import React from "react";
import { StatusSeverity } from "../types/status";

interface StatusBadgeProps {
  severity: StatusSeverity;
  label: string;
  className?: string;
}

const severityStyles: Record<StatusSeverity, { bg: string; text: string; dot: string }> = {
  success: {
    bg: "bg-emerald-50 text-emerald-700 border-emerald-200",
    text: "text-emerald-700",
    dot: "bg-emerald-500",
  },
  info: {
    bg: "bg-blue-50 text-blue-700 border-blue-200",
    text: "text-blue-700",
    dot: "bg-blue-500",
  },
  warning: {
    bg: "bg-amber-50 text-amber-700 border-amber-200",
    text: "text-amber-700",
    dot: "bg-amber-500",
  },
  error: {
    bg: "bg-rose-50 text-rose-700 border-rose-200",
    text: "text-rose-700",
    dot: "bg-rose-500",
  },
  neutral: {
    bg: "bg-zinc-100 text-zinc-700 border-zinc-200",
    text: "text-zinc-700",
    dot: "bg-zinc-400",
  },
};

export const StatusBadge: React.FC<StatusBadgeProps> = ({ severity, label, className = "" }) => {
  const style = severityStyles[severity] || severityStyles.neutral;

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border ${style.bg} ${className}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${style.dot}`} />
      <span>{label}</span>
    </span>
  );
};
