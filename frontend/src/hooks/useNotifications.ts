import { useCallback, useEffect, useRef, useState } from "react";

export type NotificationMessage = {
  text: string;
  type: "success" | "warning" | "error";
};

export function useNotifications() {
  const [message, setMessage] = useState<NotificationMessage | null>(null);
  const messageTimeoutRef = useRef<number | null>(null);

  const clearMessage = useCallback(() => {
    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
      messageTimeoutRef.current = null;
    }
    setMessage(null);
  }, []);

  const showMessage = useCallback((text: string, type: "success" | "warning" | "error" = "success") => {
    console.log(`[${type.toUpperCase()}] ${text}`);

    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
      messageTimeoutRef.current = null;
    }

    setMessage({ text, type });

    if (type !== "error") {
      messageTimeoutRef.current = window.setTimeout(() => {
        setMessage(null);
        messageTimeoutRef.current = null;
      }, type === "warning" ? 8000 : 5000);
    }
  }, []);

  useEffect(() => {
    return () => {
      if (messageTimeoutRef.current) {
        clearTimeout(messageTimeoutRef.current);
      }
    };
  }, []);

  return { message, showMessage, clearMessage };
}
