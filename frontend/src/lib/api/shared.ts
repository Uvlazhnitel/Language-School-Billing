import type { AppTransport } from "./types";

export const AUTH_REQUIRED_EVENT = "langschool:auth-required";

export class AuthRequiredError extends Error {
  constructor(message = "Authentication required") {
    super(message);
    this.name = "AuthRequiredError";
  }
}

export class AuthorizationError extends Error {
  constructor(message = "Forbidden") {
    super(message);
    this.name = "AuthorizationError";
  }
}

export class ConflictError extends Error {
  status: number;

  constructor(message = "Conflict", status = 409) {
    super(message);
    this.name = "ConflictError";
    this.status = status;
  }
}

export function isConflictError(error: unknown): error is ConflictError {
  return error instanceof ConflictError || (error instanceof Error && error.name === "ConflictError");
}

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
