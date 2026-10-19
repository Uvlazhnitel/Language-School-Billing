import { afterEach, describe, expect, it, vi } from "vitest";
import {
  newPaymentSubmission,
  submitPayment,
  type PaymentInput,
  type PaymentRequest,
} from "./paymentSubmission";
import type { PaymentDTO } from "./api";

const input: PaymentInput = { studentId: 1, amount: 10, method: "cash", note: "test" };
const payment: PaymentDTO = {
  id: 3,
  studentId: 1,
  amount: 10,
  method: "cash",
  note: "test",
  paidAt: "2026-10-19",
  createdAt: "2026-10-19",
};
const options = { uuid: () => "operation-one", today: () => "2026-10-19" };

describe("payment submission", () => {
  afterEach(() => vi.unstubAllGlobals());
  it("synchronously blocks a second click and stays locked while lists refresh", async () => {
    const state = newPaymentSubmission();
    let resolvePayment!: (value: PaymentDTO) => void;
    let resolveRefresh!: () => void;
    const send = vi.fn(
      () =>
        new Promise<PaymentDTO>((resolve) => {
          resolvePayment = resolve;
        })
    );
    const refresh = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveRefresh = resolve;
        })
    );
    const first = submitPayment(state, input, send, { ...options, afterSaved: refresh });
    expect(state.pending).toBe(true);
    expect(await submitPayment(state, input, send, options)).toBeNull();
    expect(send).toHaveBeenCalledTimes(1);
    resolvePayment(payment);
    await Promise.resolve();
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(state.pending).toBe(true);
    expect(await submitPayment(state, input, send, options)).toBeNull();
    resolveRefresh();
    expect(await first).toEqual({ payment, refreshFailed: false });
    expect(state.pending).toBe(false);
    expect(state.request).toBeNull();
  });

  it("reuses the exact request after a lost response, including after midnight", async () => {
    const state = newPaymentSubmission();
    const requests: PaymentRequest[] = [];
    const send = vi.fn(async (request: PaymentRequest) => {
      requests.push(request);
      if (requests.length === 1) throw new Error("response lost");
      return payment;
    });
    await expect(submitPayment(state, input, send, options)).rejects.toThrow("response lost");
    expect(state.pending).toBe(false);
    await submitPayment(state, input, send, {
      uuid: () => "should-not-be-used",
      today: () => "2026-10-20",
    });
    expect(requests[0]).toEqual(requests[1]);
    expect(requests[1].operationId).toBe("operation-one");
    expect(requests[1].paidAt).toBe("2026-10-19");
  });

  it.each([
    { amount: 20 },
    { studentId: 2 },
    { invoiceId: 12 },
    { method: "bank" as const },
    { note: "changed" },
  ])("creates a new operation after changing parameters: %j", async (change) => {
    const state = newPaymentSubmission();
    await expect(
      submitPayment(
        state,
        input,
        async () => {
          throw Error("failed");
        },
        options
      )
    ).rejects.toThrow();
    const send = vi.fn(async () => payment);
    await submitPayment(state, { ...input, ...change }, send, {
      ...options,
      uuid: () => "operation-two",
    });
    expect(send).toHaveBeenCalledWith(
      expect.objectContaining({ ...change, operationId: "operation-two" })
    );
  });

  it("separates refresh failures from payment failures without resending payment", async () => {
    const state = newPaymentSubmission();
    const send = vi.fn(async () => payment);
    const result = await submitPayment(state, input, send, {
      ...options,
      afterSaved: async () => {
        throw new Error("refresh failed");
      },
    });
    expect(result).toEqual({ payment, refreshFailed: true });
    expect(send).toHaveBeenCalledTimes(1);
    expect(state.pending).toBe(false);
    expect(state.request).toBeNull();
  });

  it("starts a new form with a fresh operation", async () => {
    const send = vi.fn(async () => payment);
    await submitPayment(newPaymentSubmission(), input, send, options);
    await submitPayment(newPaymentSubmission(), input, send, {
      ...options,
      uuid: () => "operation-two",
    });
    expect(send).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ operationId: "operation-one" })
    );
    expect(send).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ operationId: "operation-two" })
    );
  });
  it("generates a UUID on HTTP pages without crypto.randomUUID", async () => {
    vi.stubGlobal("crypto", { getRandomValues: (bytes: Uint8Array) => bytes.fill(17) });
    const send = vi.fn(async () => payment);
    await submitPayment(newPaymentSubmission(), input, send);
    expect(send).toHaveBeenCalledWith(
      expect.objectContaining({
        operationId: "11111111-1111-4111-9111-111111111111",
      })
    );
  });
});
