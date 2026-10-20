import { useEffect, useMemo, useSyncExternalStore } from "react";
import { ReleaseNoticeController, releaseNoticeAPI } from "../../lib/releaseNotices";
const empty = { notice: null, open: false, available: false };
const subscribeEmpty = () => () => {};
const getEmpty = () => empty;
export function useReleaseNoticeController(userId: number | undefined, active: boolean) {
  const controller = useMemo(() => {
    if (!userId || !active) return null;
    let storage: Storage | null = null;
    try {
      storage = window.localStorage;
    } catch {
      /* Private mode can block storage. */
    }
    return new ReleaseNoticeController(userId, releaseNoticeAPI, storage);
  }, [userId, active]);
  const snapshot = useSyncExternalStore(
    controller?.subscribe ?? subscribeEmpty,
    controller?.getSnapshot ?? getEmpty,
    getEmpty
  );
  useEffect(() => {
    if (!controller) return;
    void controller.start();
    const reconnect = () => {
      if (document.visibilityState !== "hidden") void controller.reconnect();
    };
    window.addEventListener("online", reconnect);
    window.addEventListener("focus", reconnect);
    document.addEventListener("visibilitychange", reconnect);
    return () => {
      controller.stop();
      window.removeEventListener("online", reconnect);
      window.removeEventListener("focus", reconnect);
      document.removeEventListener("visibilitychange", reconnect);
    };
  }, [controller]);
  useEffect(() => {
    if (!controller || !snapshot.available) return;
    const timer = window.setTimeout(
      () => controller.tick(),
      Math.min(Math.ceil(controller.remaining()), 2147483647)
    );
    return () => window.clearTimeout(timer);
  }, [controller, snapshot.available, snapshot.notice]);
  return { controller, snapshot };
}
