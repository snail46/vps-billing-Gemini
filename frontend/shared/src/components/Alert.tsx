import React from "react";
import { StatusSeverity } from "../types/status";

interface AlertProps {
  severity: StatusSeverity;
  title?: string;
  children: React.ReactNode;
  className?: string;
  onClose?: () => void;
}

const alertStyles: Record<StatusSeverity, { container: string; title: string }> = {
  success: {
    container: "bg-emerald-50 border-emerald-200 text-emerald-800",
    title: "text-emerald-900 font-semibold",
  },
  info: {
    container: "bg-blue-50 border-blue-200 text-blue-800",
    title: "text-blue-900 font-semibold",
  },
  warning: {
    container: "bg-amber-50 border-amber-200 text-amber-800",
    title: "text-amber-900 font-semibold",
  },
  error: {
    container: "bg-rose-50 border-rose-200 text-rose-800",
    title: "text-rose-900 font-semibold",
  },
  neutral: {
    container: "bg-zinc-50 border-zinc-200 text-zinc-800",
    title: "text-zinc-900 font-semibold",
  },
};

export const Alert: React.FC<AlertProps> = ({ severity, title, children, className = "", onClose }) => {
  const style = alertStyles[severity] || alertStyles.info;

  return (
    <div className={`p-4 rounded-lg border text-sm ${style.container} ${className}`} role="alert">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          {title && <div className={`mb-1 ${style.title}`}>{title}</div>}
          <div>{children}</div>
        </div>
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200 p-0.5 rounded cursor-pointer transition-colors leading-none"
            aria-label="Close"
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
};
