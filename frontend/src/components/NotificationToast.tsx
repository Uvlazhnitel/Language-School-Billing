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
  const toastStyle = { backgroundColor };

  return (
    <div
      className={`messageToast ${message.type}`}
      style={toastStyle}
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
          className="messageToastClose"
        >
          ×
        </button>
      </div>
    </div>
  );
}
