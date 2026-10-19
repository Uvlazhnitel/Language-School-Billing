import type { PaymentDTO, PaymentMethod } from "./api";

export type PaymentInput = {
  studentId: number;
  invoiceId?: number;
  amount: number;
  method: PaymentMethod;
  note: string;
};
export type PaymentRequest = PaymentInput & { operationId: string; paidAt: string };
export type PaymentSubmissionState = {
  pending: boolean;
  request: PaymentRequest | null;
};

export function newPaymentSubmission(): PaymentSubmissionState {
  return { pending: false, request: null };
}

// randomUUID is restricted to secure contexts; direct HTTP deployments can
// still generate a cryptographic UUID using getRandomValues.
function newOperationId(): string {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

// Keep the request (including its date) after ambiguous failures, so a retry
// cannot accidentally record the same payment as a second operation.
export async function submitPayment(
  state: PaymentSubmissionState,
  input: PaymentInput,
  send: (request: PaymentRequest) => Promise<PaymentDTO>,
  options: {
    uuid?: () => string;
    today?: () => string;
    afterSaved?: (payment: PaymentDTO) => Promise<void>;
  } = {}
): Promise<{ payment: PaymentDTO; refreshFailed: boolean } | null> {
  const uuid = options.uuid ?? newOperationId;
  const today = options.today ?? (() => new Date().toISOString().slice(0, 10));
  if (state.pending) return null;
  state.pending = true;
  try {
    const previous = state.request;
    if (
      !previous ||
      previous.studentId !== input.studentId ||
      previous.invoiceId !== input.invoiceId ||
      previous.amount !== input.amount ||
      previous.method !== input.method ||
      previous.note !== input.note
    ) {
      state.request = { ...input, operationId: uuid(), paidAt: today() };
    }
    const result = await send(state.request!);
    state.request = null;
    try {
      await options.afterSaved?.(result);
      return { payment: result, refreshFailed: false };
    } catch {
      return { payment: result, refreshFailed: true };
    }
  } finally {
    state.pending = false;
  }
}
