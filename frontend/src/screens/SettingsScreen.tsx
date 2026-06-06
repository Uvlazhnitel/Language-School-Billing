import type { UserDTO } from "../lib/api";
import type { TranslateFn, UiLocale } from "../lib/i18n";
import type { AppTabId } from "../lib/appUi";
import { canShowInvoiceFolderAction, canShowSettingsFilesCard } from "../lib/uiCapabilities";

type UserDraft = { username: string; role: string; isActive: boolean };

type SettingsScreenProps = {
  uiLocale: UiLocale;
  canManageSettings: boolean;
  canCreateBackups: boolean;
  creatingBackup: boolean;
  transportCapabilities: {
    isDesktop: boolean;
    canOpenLocalFiles: boolean;
    canOpenFolders: boolean;
    canDownloadPdf: boolean;
  };
  appDirs: Record<string, string> | null;
  canManageUsers: boolean;
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
  onOpenAppFolder: (path: string | undefined, label: string) => void | Promise<void>;
  onSetTab: (tab: AppTabId) => void;
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
  canManageSettings,
  canCreateBackups,
  creatingBackup,
  transportCapabilities,
  appDirs,
  canManageUsers,
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
  onOpenAppFolder,
  onSetTab,
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
            disabled={!canManageSettings}
            onChange={(e) => void onLocaleChange(e.target.value as UiLocale)}
          >
            <option value="en-US">{t("settings.languageEnglish")}</option>
            <option value="ru-RU">{t("settings.languageRussian")}</option>
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
          {canShowInvoiceFolderAction(transportCapabilities) && (
            <button
              type="button"
              className="workspaceActionButton"
              onClick={() => void onOpenAppFolder(appDirs?.backups, t("field.backups").toLowerCase())}
              disabled={!appDirs?.backups}
            >
              {t("button.backupsFolder")}
            </button>
          )}
        </div>
      </section>

      {canShowSettingsFilesCard(transportCapabilities) && (
        <section className="detailCard">
          <div className="detailCardHeader">
            <h3>{t("settings.filesTitle")}</h3>
          </div>
          <p className="mutedInline">{t("settings.filesDesc")}</p>
          <div className="settingsActions">
            <button
              type="button"
              className="workspaceActionButton"
              onClick={() => void onOpenAppFolder(appDirs?.invoices, t("tabs.invoice").toLowerCase())}
              disabled={!appDirs?.invoices}
            >
              {t("button.invoicesFolder")}
            </button>
            <button
              type="button"
              className="workspaceActionButton"
              onClick={() => void onOpenAppFolder(appDirs?.exports, "exports")}
              disabled={!appDirs?.exports}
            >
              {t("button.exportsFolder")}
            </button>
            <button
              type="button"
              className="workspaceActionButton"
              onClick={() => void onOpenAppFolder(appDirs?.data, "data")}
              disabled={!appDirs?.data}
            >
              {t("button.dataFolder")}
            </button>
          </div>
        </section>
      )}

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

      {canManageUsers && (
        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>Users</h3>
          </div>
          <p className="mutedInline">Manage admin and staff accounts for the web app.</p>

          <div className="formRow">
            <label>Username</label>
            <input value={newUserUsername} onChange={(e) => onNewUserUsernameChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>Password</label>
            <input
              type="password"
              value={newUserPassword}
              onChange={(e) => onNewUserPasswordChange(e.target.value)}
            />
          </div>
          <div className="formRow">
            <label>Role</label>
            <select value={newUserRole} onChange={(e) => onNewUserRoleChange(e.target.value)}>
              <option value="staff">staff</option>
              <option value="admin">admin</option>
            </select>
          </div>
          <div className="settingsActions">
            <button
              type="button"
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={() => void onCreateUser()}
              disabled={creatingUser}
            >
              {creatingUser ? "Create..." : "Create user"}
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
                    <th>Username</th>
                    <th>Role</th>
                    <th>Active</th>
                    <th>Password reset</th>
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
                            <option value="staff">staff</option>
                            <option value="admin">admin</option>
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
                            placeholder="New password"
                          />
                        </td>
                        <td>
                          <button onClick={() => void onSaveUser(user.id)}>{t("button.save")}</button>
                          <button onClick={() => void onResetUserPassword(user.id)}>Reset password</button>
                          <button
                            onClick={() => void onDeleteUser(user.id)}
                            disabled={currentSessionUser?.id === user.id}
                          >
                            Delete
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
