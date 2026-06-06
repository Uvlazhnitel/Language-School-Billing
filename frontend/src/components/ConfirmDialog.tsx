import type { TranslateFn } from "../lib/i18n";
import type { ConfirmDialogState } from "../hooks/useConfirmDialog";

type ConfirmDialogProps = {
  dialog: ConfirmDialogState;
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
  t: TranslateFn;
};

export function ConfirmDialog({ dialog, onConfirm, onCancel, t }: ConfirmDialogProps) {
  return (
    <div
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: "rgba(0, 0, 0, 0.5)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 10100,
      }}
    >
      <div
        style={{
          backgroundColor: "white",
          padding: "24px",
          borderRadius: "8px",
          maxWidth: "500px",
          boxShadow: "0 4px 16px rgba(0,0,0,0.3)",
        }}
      >
        <h3 style={{ marginTop: 0, marginBottom: "16px" }}>{t("modal.confirm")}</h3>
        <p style={{ marginBottom: "24px", lineHeight: "1.5" }}>{dialog.message}</p>
        <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
          <button onClick={onCancel} style={{ padding: "8px 16px" }}>
            {t("button.cancel")}
          </button>
          <button
            onClick={() => void onConfirm()}
            style={{
              padding: "8px 16px",
              backgroundColor: "#f44336",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            {dialog.confirmButtonLabel ?? t("msg.confirmDelete")}
          </button>
        </div>
      </div>
    </div>
  );
}
