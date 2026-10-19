import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { PaymentModal } from "./PaymentModal";
import { createTranslator } from "../../lib/i18n";

describe("payment modal", () => {
  const props = {
    studentId: 1,
    studentName: "Student",
    invoiceId: null,
    amount: "10",
    note: "",
    onAmountChange: vi.fn(),
    onNoteChange: vi.fn(),
    onCancel: vi.fn(),
    onSubmit: vi.fn(),
  };
  it.each(["en-US", "ru-RU", "lv-LV"] as const)(
    "disables inputs and both actions while saving (%s)",
    (locale) => {
      const t = createTranslator(locale);
      const html = renderToStaticMarkup(<PaymentModal {...props} saving t={t} />);
      expect(html).toContain('aria-busy="true"');
      expect(html.match(/disabled=""/g)).toHaveLength(5);
      expect(html).toContain(t("button.savingPayment"));
      expect(html).not.toContain("button.savingPayment");
    }
  );
  it("enables editing and submitting when idle", () => {
    const html = renderToStaticMarkup(
      <PaymentModal {...props} saving={false} t={createTranslator("en-US")} />
    );
    expect(html.match(/disabled=""/g)).toHaveLength(1);
    expect(html).toContain("Record payment");
  });
  it("ignores backdrop clicks while saving and closes when idle", () => {
    const onCancel = vi.fn();
    const t = createTranslator("en-US");
    PaymentModal({ ...props, saving: true, onCancel, t })?.props.onClick();
    expect(onCancel).not.toHaveBeenCalled();
    PaymentModal({ ...props, saving: false, onCancel, t })?.props.onClick();
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
