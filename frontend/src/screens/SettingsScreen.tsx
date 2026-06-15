import type { InvoiceEmailSettingsDTO, UserDTO } from "../lib/api";
import type { TranslateFn, UiLocale } from "../lib/i18n";
import type { AppTabId } from "../lib/appUi";

type UserDraft = { username: string; role: string; isActive: boolean };

type SettingsScreenProps = {
  uiLocale: UiLocale;
  canCreateBackups: boolean;
  canManageSettings: boolean;
  creatingBackup: boolean;
  canManageUsers: boolean;
  invoiceEmailSettingsLoading: boolean;
  savingInvoiceEmailSettings: boolean;
  invoiceEmailSettings: InvoiceEmailSettingsDTO | null;
  invoiceEmailSubjectTemplate: string;
  invoiceEmailBodyTemplate: string;
  invoiceEmailReplyTo: string;
  usersLoading: boolean;
  users: UserDTO[];
  creatingUser: boolean;
  newUserUsername: string;
  newUserPassword: string;
  newUserRole: string;
  userDrafts: Record<number, UserDraft>;
  userPasswordDrafts: Record<number, string>;
  currentSessionUser: { id: number; username: string; role: string } | null;
  onLocaleChange: (value: UiLocale) => void | Promise<void>;
  onCreateBackup: () => void | Promise<void>;
  onSetTab: (tab: AppTabId) => void;
  onInvoiceEmailSubjectTemplateChange: (value: string) => void;
  onInvoiceEmailBodyTemplateChange: (value: string) => void;
  onInvoiceEmailReplyToChange: (value: string) => void;
  onSaveInvoiceEmailSettings: () => void | Promise<void>;
  onResetInvoiceEmailSettings: () => void | Promise<void>;
  onNewUserUsernameChange: (value: string) => void;
  onNewUserPasswordChange: (value: string) => void;
  onNewUserRoleChange: (value: string) => void;
  onCreateUser: () => void | Promise<void>;
  onRefreshUsers: () => void | Promise<void>;
  onUserDraftsChange: (updater: (prev: Record<number, UserDraft>) => Record<number, UserDraft>) => void;
  onUserPasswordDraftsChange: (updater: (prev: Record<number, string>) => Record<number, string>) => void;
  onSaveUser: (userId: number) => void | Promise<void>;
  onResetUserPassword: (userId: number) => void | Promise<void>;
  onDeleteUser: (userId: number) => void | Promise<void>;
  t: TranslateFn;
};

export function SettingsScreen({
  uiLocale,
  canCreateBackups,
  canManageSettings,
  creatingBackup,
  canManageUsers,
  invoiceEmailSettingsLoading,
  savingInvoiceEmailSettings,
  invoiceEmailSettings,
  invoiceEmailSubjectTemplate,
  invoiceEmailBodyTemplate,
  invoiceEmailReplyTo,
  usersLoading,
  users,
  creatingUser,
  newUserUsername,
  newUserPassword,
  newUserRole,
  userDrafts,
  userPasswordDrafts,
  currentSessionUser,
  onLocaleChange,
  onCreateBackup,
  onSetTab,
  onInvoiceEmailSubjectTemplateChange,
  onInvoiceEmailBodyTemplateChange,
  onInvoiceEmailReplyToChange,
  onSaveInvoiceEmailSettings,
  onResetInvoiceEmailSettings,
  onNewUserUsernameChange,
  onNewUserPasswordChange,
  onNewUserRoleChange,
  onCreateUser,
  onRefreshUsers,
  onUserDraftsChange,
  onUserPasswordDraftsChange,
  onSaveUser,
  onResetUserPassword,
  onDeleteUser,
  t,
}: SettingsScreenProps) {
  return (
    <div className="settingsGrid">
      <section className="detailCard">
        <div className="detailCardHeader">
          <h3>{t("settings.languageTitle")}</h3>
        </div>
        <p className="mutedInline">{t("settings.languageDesc")}</p>
        <div className="formRow">
          <label>{t("settings.locale")}</label>
          <select
            value={uiLocale}
            onChange={(e) => void onLocaleChange(e.target.value as UiLocale)}
          >
            <option value="en-US">{t("settings.languageEnglish")}</option>
            <option value="ru-RU">{t("settings.languageRussian")}</option>
            <option value="lv-LV">{t("settings.languageLatvian")}</option>
          </select>
        </div>
      </section>

      <section className="detailCard">
        <div className="detailCardHeader">
          <h3>{t("settings.backupsTitle")}</h3>
        </div>
        <p className="mutedInline">{t("settings.backupDesc")}</p>
        <div className="settingsActions">
          <button
            type="button"
            className="workspaceActionButton workspaceActionButtonPrimary"
            onClick={() => void onCreateBackup()}
            disabled={creatingBackup || !canCreateBackups}
          >
            {creatingBackup ? `${t("button.createBackup")}...` : t("button.createBackup")}
          </button>
        </div>
      </section>

      <section className="detailCard">
        <div className="detailCardHeader">
          <h3>{t("button.filesAndCopies")}</h3>
        </div>
        <p className="mutedInline">{t("msg.systemSectionsNav")}</p>
        <div className="settingsActions">
          <button type="button" className="workspaceActionButton" onClick={() => onSetTab("invoice")}>
            {t("tabs.invoice")}
          </button>
        </div>
      </section>

      {canManageSettings && (
        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("settings.invoiceEmailTitle")}</h3>
          </div>
          <p className="mutedInline">{t("settings.invoiceEmailDesc")}</p>
          {invoiceEmailSettingsLoading && !invoiceEmailSettings ? (
            <div className="empty">{t("label.loading")}</div>
          ) : (
            <>
              <div className="formRow">
                <label>{t("settings.invoiceEmailSubject")}</label>
                <input
                  value={invoiceEmailSubjectTemplate}
                  onChange={(e) => onInvoiceEmailSubjectTemplateChange(e.target.value)}
                />
              </div>
              <div className="formRow formRowTopAligned">
                <label>{t("settings.invoiceEmailBody")}</label>
                <textarea
                  className="modalTextarea"
                  rows={9}
                  value={invoiceEmailBodyTemplate}
                  onChange={(e) => onInvoiceEmailBodyTemplateChange(e.target.value)}
                />
              </div>
              <div className="formRow">
                <label>{t("settings.invoiceEmailReplyTo")}</label>
                <input
                  value={invoiceEmailReplyTo}
                  onChange={(e) => onInvoiceEmailReplyToChange(e.target.value)}
                  placeholder={t("settings.invoiceEmailReplyToPlaceholder")}
                />
              </div>
              <div className="formRow formRowTopAligned">
                <label>{t("settings.invoiceEmailPlaceholders")}</label>
                <div className="templatePlaceholderList">
                  {(invoiceEmailSettings?.availablePlaceholders ?? []).map((placeholder) => (
                    <code key={placeholder} className="templatePlaceholderTag">
                      {placeholder}
                    </code>
                  ))}
                </div>
              </div>
              <div className="settingsActions">
                <button
                  type="button"
                  className="workspaceActionButton workspaceActionButtonPrimary"
                  onClick={() => void onSaveInvoiceEmailSettings()}
                  disabled={savingInvoiceEmailSettings}
                >
                  {savingInvoiceEmailSettings ? `${t("button.save")}...` : t("button.save")}
                </button>
                <button
                  type="button"
                  className="workspaceActionButton"
                  onClick={() => void onResetInvoiceEmailSettings()}
                  disabled={savingInvoiceEmailSettings}
                >
                  {t("settings.invoiceEmailReset")}
                </button>
              </div>
            </>
          )}
        </section>
      )}

      {canManageUsers && (
        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("settings.usersTitle")}</h3>
          </div>
          <p className="mutedInline">{t("settings.usersDesc")}</p>

          <div className="formRow">
            <label>{t("settings.userUsername")}</label>
            <input value={newUserUsername} onChange={(e) => onNewUserUsernameChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>{t("settings.userPassword")}</label>
            <input
              type="password"
              value={newUserPassword}
              onChange={(e) => onNewUserPasswordChange(e.target.value)}
            />
          </div>
          <div className="formRow">
            <label>{t("settings.userRole")}</label>
            <select value={newUserRole} onChange={(e) => onNewUserRoleChange(e.target.value)}>
              <option value="staff">{t("settings.userRoleStaff")}</option>
              <option value="admin">{t("settings.userRoleAdmin")}</option>
            </select>
          </div>
          <div className="settingsActions">
            <button
              type="button"
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={() => void onCreateUser()}
              disabled={creatingUser}
            >
              {creatingUser ? t("settings.userCreatePending") : t("settings.userCreate")}
            </button>
            <button type="button" className="workspaceActionButton" onClick={() => void onRefreshUsers()}>
              {t("button.refresh")}
            </button>
          </div>

          {usersLoading ? (
            <div className="empty">{t("label.loading")}</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>{t("settings.userUsername")}</th>
                    <th>{t("settings.userRole")}</th>
                    <th>{t("settings.userActive")}</th>
                    <th>{t("settings.userPasswordReset")}</th>
                    <th>{t("field.actions")}</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => {
                    const draft = userDrafts[user.id] ?? {
                      username: user.username,
                      role: user.role,
                      isActive: user.isActive,
                    };
                    return (
                      <tr key={user.id}>
                        <td>
                          <input
                            value={draft.username}
                            onChange={(e) =>
                              onUserDraftsChange((prev) => ({
                                ...prev,
                                [user.id]: { ...draft, username: e.target.value },
                              }))
                            }
                          />
                        </td>
                        <td>
                          <select
                            value={draft.role}
                            onChange={(e) =>
                              onUserDraftsChange((prev) => ({
                                ...prev,
                                [user.id]: { ...draft, role: e.target.value },
                              }))
                            }
                          >
                            <option value="staff">{t("settings.userRoleStaff")}</option>
                            <option value="admin">{t("settings.userRoleAdmin")}</option>
                          </select>
                        </td>
                        <td>
                          <input
                            type="checkbox"
                            checked={draft.isActive}
                            onChange={(e) =>
                              onUserDraftsChange((prev) => ({
                                ...prev,
                                [user.id]: { ...draft, isActive: e.target.checked },
                              }))
                            }
                          />
                        </td>
                        <td>
                          <input
                            type="password"
                            value={userPasswordDrafts[user.id] ?? ""}
                            onChange={(e) =>
                              onUserPasswordDraftsChange((prev) => ({
                                ...prev,
                                [user.id]: e.target.value,
                              }))
                            }
                            placeholder={t("settings.userPasswordPlaceholder")}
                          />
                        </td>
                        <td>
                          <button onClick={() => void onSaveUser(user.id)}>{t("button.save")}</button>
                          <button onClick={() => void onResetUserPassword(user.id)}>
                            {t("settings.userResetPassword")}
                          </button>
                          <button
                            onClick={() => void onDeleteUser(user.id)}
                            disabled={currentSessionUser?.id === user.id}
                          >
                            {t("settings.userDelete")}
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
