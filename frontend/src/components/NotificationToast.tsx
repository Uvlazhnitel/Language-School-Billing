import type { NotificationMessage } from "../hooks/useNotifications";
import type { TranslateFn } from "../lib/i18n";

type NotificationToastProps = {
  message: NotificationMessage;
  onDismiss: () => void;
  t: TranslateFn;
};

export function NotificationToast({ message, onDismiss, t }: NotificationToastProps) {
  const backgroundColor =
    message.type === "success" ? "#4caf50" : message.type === "warning" ? "#f59e0b" : "#f44336";
  const isAlert = message.type === "error" || message.type === "warning";

  return (
    <div
      className={`messageToast ${message.type}`}
      style={{
        position: "fixed",
        top: "20px",
        right: "20px",
        padding: "16px 24px",
        backgroundColor,
        color: "white",
        borderRadius: "4px",
        boxShadow: "0 2px 8px rgba(0,0,0,0.2)",
        zIndex: 10000,
        maxWidth: "400px",
        fontSize: "14px",
        lineHeight: "1.5",
      }}
      role={isAlert ? "alert" : "status"}
      aria-live={isAlert ? "assertive" : "polite"}
      onClick={onDismiss}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: "12px",
        }}
      >
        <span>{message.text}</span>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDismiss();
          }}
          aria-label={t("msg.closeNotification")}
          style={{
            background: "none",
            border: "none",
            color: "white",
            cursor: "pointer",
            fontSize: "18px",
            padding: "0",
            lineHeight: "1",
          }}
        >
          ×
        </button>
      </div>
    </div>
  );
}
