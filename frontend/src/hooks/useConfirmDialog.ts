import { useCallback, useState } from "react";

export type ConfirmDialogState = {
  isOpen: boolean;
  message: string;
  onConfirm: () => void | Promise<void>;
  confirmButtonLabel?: string;
};

export function useConfirmDialog() {
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState | null>(null);

  const showConfirm = useCallback(
    (
      messageText: string,
      onConfirm: () => void | Promise<void>,
      confirmButtonLabel?: string
    ) => {
      setConfirmDialog({
        isOpen: true,
        message: messageText,
        onConfirm,
        confirmButtonLabel,
      });
    },
    []
  );

  const handleConfirmYes = useCallback(async () => {
    try {
      if (confirmDialog?.onConfirm) {
        await confirmDialog.onConfirm();
      }
    } finally {
      setConfirmDialog(null);
    }
  }, [confirmDialog]);

  const handleConfirmNo = useCallback(() => {
    setConfirmDialog(null);
  }, []);

  return { confirmDialog, showConfirm, handleConfirmYes, handleConfirmNo };
}
