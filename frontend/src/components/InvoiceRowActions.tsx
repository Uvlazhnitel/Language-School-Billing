import type { ReactNode } from "react";
import type { InvoiceListItemView } from "../lib/invoices";
import type { TranslateFn } from "../lib/i18n";
import { getInvoiceRowActions, type InvoiceRowAction } from "../lib/invoiceUi";

type Props = {
  invoice: InvoiceListItemView;
  menu: ReactNode;
  onAction: (action: InvoiceRowAction, invoiceId: number) => void;
  t: TranslateFn;
};

export function InvoiceRowActions({ invoice, menu, onAction, t }: Props) {
  const { primary } = getInvoiceRowActions(invoice);
  const label = primary === "recordPayment" ? "button.recordPayment" : `button.${primary}`;
  return (
    <div className="invoiceListActions">
      <button
        type="button"
        className="workspaceActionButton workspaceActionButtonPrimary invoiceListPrimary"
        onClick={() => onAction(primary, invoice.id)}
      >
        {t(label)}
      </button>
      <div className="invoiceListMenuSlot">{menu}</div>
    </div>
  );
}
