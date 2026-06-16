import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { SettingsScreen } from "./SettingsScreen";
import { createTranslator } from "../lib/i18n";

describe("SettingsScreen", () => {
  it("shows invoice email settings block for admins", () => {
    const markup = renderToStaticMarkup(
      <SettingsScreen
        uiLocale="lv-LV"
        canCreateBackups
        canManageSettings
        canViewInvoiceArchive
        creatingBackup={false}
        canManageUsers={false}
        invoiceArchiveLoading={false}
        invoiceArchive={{
          years: [
            {
              year: 2026,
              months: [
                {
                  month: 6,
                  files: [
                    {
                      filename: "LS-202606-001.pdf",
                      year: 2026,
                      month: 6,
                      openUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/open",
                      downloadUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/download",
                    },
                  ],
                },
              ],
            },
          ],
        }}
        invoiceEmailSettingsLoading={false}
        savingInvoiceEmailSettings={false}
        invoiceEmailSettings={{
          subjectTemplate: "Rēķins {invoice_number}",
          bodyTemplate: "Labdien!",
          replyTo: "",
          availablePlaceholders: ["{invoice_number}", "{amount}"],
        }}
        invoiceEmailSubjectTemplate="Rēķins {invoice_number}"
        invoiceEmailBodyTemplate="Labdien!"
        invoiceEmailReplyTo=""
        usersLoading={false}
        users={[]}
        creatingUser={false}
        newUserUsername=""
        newUserPassword=""
        newUserRole="staff"
        userDrafts={{}}
        userPasswordDrafts={{}}
        currentSessionUser={null}
        onLocaleChange={vi.fn()}
        onCreateBackup={vi.fn()}
        onRefreshInvoiceArchive={vi.fn()}
        onSetTab={vi.fn()}
        onInvoiceEmailSubjectTemplateChange={vi.fn()}
        onInvoiceEmailBodyTemplateChange={vi.fn()}
        onInvoiceEmailReplyToChange={vi.fn()}
        onSaveInvoiceEmailSettings={vi.fn()}
        onResetInvoiceEmailSettings={vi.fn()}
        onNewUserUsernameChange={vi.fn()}
        onNewUserPasswordChange={vi.fn()}
        onNewUserRoleChange={vi.fn()}
        onCreateUser={vi.fn()}
        onRefreshUsers={vi.fn()}
        onUserDraftsChange={vi.fn()}
        onUserPasswordDraftsChange={vi.fn()}
        onSaveUser={vi.fn()}
        onResetUserPassword={vi.fn()}
        onDeleteUser={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Invoice email templates");
    expect(markup).toContain("Invoice archive");
    expect(markup).toContain("LS-202606-001.pdf");
    expect(markup).toContain("{invoice_number}");
    expect(markup).toContain("Reset to default");
  });
});
