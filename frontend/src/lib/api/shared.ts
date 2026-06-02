import type { AppTransport } from "./types";

declare global {
  interface Window {
    go?: {
      main?: {
        App?: unknown;
      };
    };
    runtime?: unknown;
  }
}

export function isWailsRuntime(): boolean {
  if (typeof window === "undefined") return false;
  return Boolean(window.go?.main?.App || window.runtime);
}

let transportInstance: AppTransport | null = null;

export function setTransportForTests(transport: AppTransport | null) {
  transportInstance = transport;
}

export async function getTransport(): Promise<AppTransport> {
  if (transportInstance) return transportInstance;

  if (isWailsRuntime()) {
    const mod = await import("./wailsTransport");
    transportInstance = mod.wailsTransport;
    return transportInstance;
  }

  const mod = await import("./httpTransport");
  transportInstance = mod.httpTransport;
  return transportInstance;
}
