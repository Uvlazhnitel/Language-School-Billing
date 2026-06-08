import type { ReactNode } from "react";

type EmptyStateProps = {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  compact?: boolean;
  children?: ReactNode;
};

export function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
  secondaryActionLabel,
  onSecondaryAction,
  compact = false,
  children,
}: EmptyStateProps) {
  return (
    <div className={`empty emptyState ${compact ? "emptyState--compact" : ""}`}>
      <div className="emptyStateCopy">
        <strong>{title}</strong>
        <p>{description}</p>
      </div>
      {(actionLabel && onAction) || (secondaryActionLabel && onSecondaryAction) ? (
        <div className="emptyStateActions">
          {actionLabel && onAction ? (
            <button className="btn btnPrimary" onClick={onAction}>
              {actionLabel}
            </button>
          ) : null}
          {secondaryActionLabel && onSecondaryAction ? (
            <button className="secondaryActionButton" onClick={onSecondaryAction}>
              {secondaryActionLabel}
            </button>
          ) : null}
        </div>
      ) : null}
      {children}
    </div>
  );
}
