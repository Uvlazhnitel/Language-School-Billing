import type { NotificationMessage } from "../hooks/useNotifications";
import type { TranslateFn } from "../lib/i18n";

type NotificationToastProps = {
  message: NotificationMessage;
  onDismiss: () => void;
  t: TranslateFn;
};

export function NotificationToast({ message, onDismiss, t }: NotificationToastProps) {
  return (
    <div
      className={`messageToast ${message.type}`}
      style={{
        position: "fixed",
        top: "20px",
        right: "20px",
        padding: "16px 24px",
        backgroundColor: message.type === "success" ? "#4caf50" : "#f44336",
        color: "white",
        borderRadius: "4px",
        boxShadow: "0 2px 8px rgba(0,0,0,0.2)",
        zIndex: 10000,
        maxWidth: "400px",
        fontSize: "14px",
        lineHeight: "1.5",
      }}
      role={message.type === "error" ? "alert" : "status"}
      aria-live={message.type === "error" ? "assertive" : "polite"}
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
