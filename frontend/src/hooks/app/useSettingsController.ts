import { useCallback, useEffect, useState } from "react";

import { getTransport, type InvoiceArchiveResult, type InvoiceEmailSettingsDTO, type UserDTO } from "../../lib/api";
import { createTranslator, type TranslateFn, type UiLocale } from "../../lib/i18n";

type UserDraft = { username: string; role: string; isActive: boolean };

type UseSettingsControllerParams = {
  appReady: boolean;
  isAuthenticated: boolean;
  tab: string;
  canManageUsers: boolean;
  canManageSettings: boolean;
  canCreateBackups: boolean;
  canViewInvoiceArchive: boolean;
  uiLocale: UiLocale;
  setUiLocale: (locale: UiLocale) => void;
  showMessage: (message: string, type?: "success" | "error") => void;
  showConfirm: (message: string, onConfirm: () => void | Promise<void>, confirmLabel?: string) => void;
  t: TranslateFn;
};

export function useSettingsController({
  appReady,
  isAuthenticated,
  tab,
  canManageUsers,
  canManageSettings,
  canCreateBackups,
  canViewInvoiceArchive,
  uiLocale,
  setUiLocale,
  showMessage,
  showConfirm,
  t,
}: UseSettingsControllerParams) {
  const [creatingBackup, setCreatingBackup] = useState(false);
  const [invoiceArchive, setInvoiceArchive] = useState<InvoiceArchiveResult | null>(null);
  const [invoiceArchiveLoading, setInvoiceArchiveLoading] = useState(false);
  const [invoiceEmailSettings, setInvoiceEmailSettings] = useState<InvoiceEmailSettingsDTO | null>(null);
  const [invoiceEmailSettingsLoading, setInvoiceEmailSettingsLoading] = useState(false);
  const [savingInvoiceEmailSettings, setSavingInvoiceEmailSettings] = useState(false);
  const [invoiceEmailSubjectTemplate, setInvoiceEmailSubjectTemplate] = useState("");
  const [invoiceEmailBodyTemplate, setInvoiceEmailBodyTemplate] = useState("");
  const [invoiceEmailReplyTo, setInvoiceEmailReplyTo] = useState("");
  const [users, setUsers] = useState<UserDTO[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [creatingUser, setCreatingUser] = useState(false);
  const [newUserUsername, setNewUserUsername] = useState("");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [newUserRole, setNewUserRole] = useState("staff");
  const [userDrafts, setUserDrafts] = useState<Record<number, UserDraft>>({});
  const [userPasswordDrafts, setUserPasswordDrafts] = useState<Record<number, string>>({});

  const createManualBackup = useCallback(async () => {
    if (!canCreateBackups) return;
    try {
      setCreatingBackup(true);
      const transport = await getTransport();
      const backup = await transport.createBackup();
      showMessage(t("msg.backupCreated", { path: backup.path ?? backup.filename }));
    } catch (e: any) {
      showMessage(t("msg.backupCreateError", { message: String(e?.message ?? e) }), "error");
    } finally {
      setCreatingBackup(false);
    }
  }, [canCreateBackups, showMessage, t]);

  const loadInvoiceArchive = useCallback(async () => {
    if (!canViewInvoiceArchive) return;
    try {
      setInvoiceArchiveLoading(true);
      const transport = await getTransport();
      const archive = await transport.listInvoiceArchive();
      setInvoiceArchive(archive);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setInvoiceArchiveLoading(false);
    }
  }, [canViewInvoiceArchive, showMessage, t]);

  const loadUsers = useCallback(async () => {
    if (!canManageUsers) return;
    try {
      setUsersLoading(true);
      const transport = await getTransport();
      const items = await transport.listUsers();
      setUsers(items);
      setUserDrafts(
        Object.fromEntries(
          items.map((item) => [
            item.id,
            { username: item.username, role: item.role, isActive: item.isActive },
          ]),
        ),
      );
    } catch (e: any) {
      showMessage(t("msg.userLoadError", { message: String(e?.message ?? e) }), "error");
    } finally {
      setUsersLoading(false);
    }
  }, [canManageUsers, showMessage, t]);

  useEffect(() => {
    if (appReady && isAuthenticated && canManageUsers && tab === "settings") {
      void loadUsers();
    }
  }, [appReady, isAuthenticated, canManageUsers, loadUsers, tab]);

  useEffect(() => {
    if (!appReady || tab !== "settings" || !canViewInvoiceArchive) return;
    void loadInvoiceArchive();
  }, [appReady, canViewInvoiceArchive, loadInvoiceArchive, tab]);

  const handleCreateUser = useCallback(async () => {
    try {
      setCreatingUser(true);
      const transport = await getTransport();
      const created = await transport.createUser(newUserUsername, newUserPassword, newUserRole);
      setUsers((prev) => [...prev, created]);
      setUserDrafts((prev) => ({
        ...prev,
        [created.id]: {
          username: created.username,
          role: created.role,
          isActive: created.isActive,
        },
      }));
      setNewUserUsername("");
      setNewUserPassword("");
      setNewUserRole("staff");
      showMessage(t("msg.userCreated"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setCreatingUser(false);
    }
  }, [newUserPassword, newUserRole, newUserUsername, showMessage, t]);

  const handleSaveUser = useCallback(async (userId: number) => {
    const draft = userDrafts[userId];
    if (!draft) return;
    try {
      const transport = await getTransport();
      const updated = await transport.updateUser(userId, draft.username, draft.role, draft.isActive);
      setUsers((prev) => prev.map((item) => (item.id === userId ? updated : item)));
      setUserDrafts((prev) => ({
        ...prev,
        [userId]: { username: updated.username, role: updated.role, isActive: updated.isActive },
      }));
      showMessage(t("msg.userUpdated"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [showMessage, t, userDrafts]);

  const handleDeleteUser = useCallback(async (userId: number) => {
    const target = users.find((item) => item.id === userId);
    if (!target) return;
    showConfirm(
      t("msg.userDeleteConfirm", { username: target.username }),
      async () => {
        try {
          const transport = await getTransport();
          await transport.deleteUser(userId);
          setUsers((prev) => prev.filter((item) => item.id !== userId));
          setUserDrafts((prev) => {
            const next = { ...prev };
            delete next[userId];
            return next;
          });
          setUserPasswordDrafts((prev) => {
            const next = { ...prev };
            delete next[userId];
            return next;
          });
          showMessage(t("msg.userDeleted"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      },
      t("settings.userDelete"),
    );
  }, [showConfirm, showMessage, t, users]);

  const handleResetUserPassword = useCallback(async (userId: number) => {
    const password = userPasswordDrafts[userId]?.trim() ?? "";
    if (!password) {
      showMessage(t("msg.userPasswordRequired"), "error");
      return;
    }
    try {
      const transport = await getTransport();
      await transport.setUserPassword(userId, password);
      setUserPasswordDrafts((prev) => ({ ...prev, [userId]: "" }));
      showMessage(t("msg.userPasswordReset"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [showMessage, t, userPasswordDrafts]);

  const handleLocaleChange = useCallback(async (nextLocale: UiLocale) => {
    const previousLocale = uiLocale;
    setUiLocale(nextLocale);
    try {
      const transport = await getTransport();
      await transport.setLocale(nextLocale);
      showMessage(createTranslator(nextLocale)("settings.languageSaved"));
    } catch (e: any) {
      setUiLocale(previousLocale);
      showMessage(
        createTranslator(previousLocale)("settings.languageSaveError") + `: ${String(e?.message ?? e)}`,
        "error",
      );
    }
  }, [setUiLocale, showMessage, uiLocale]);

  const applyInvoiceEmailSettingsDraft = useCallback((settings: InvoiceEmailSettingsDTO) => {
    setInvoiceEmailSettings(settings);
    setInvoiceEmailSubjectTemplate(settings.subjectTemplate);
    setInvoiceEmailBodyTemplate(settings.bodyTemplate);
    setInvoiceEmailReplyTo(settings.replyTo);
  }, []);

  const loadInvoiceEmailSettings = useCallback(async () => {
    if (!canManageSettings) return;
    setInvoiceEmailSettingsLoading(true);
    try {
      const transport = await getTransport();
      const settings = await transport.getInvoiceEmailSettings();
      applyInvoiceEmailSettingsDraft(settings);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setInvoiceEmailSettingsLoading(false);
    }
  }, [applyInvoiceEmailSettingsDraft, canManageSettings, showMessage, t]);

  const handleSaveInvoiceEmailSettings = useCallback(async () => {
    setSavingInvoiceEmailSettings(true);
    try {
      const transport = await getTransport();
      const settings = await transport.saveInvoiceEmailSettings({
        subjectTemplate: invoiceEmailSubjectTemplate,
        bodyTemplate: invoiceEmailBodyTemplate,
        replyTo: invoiceEmailReplyTo,
      });
      applyInvoiceEmailSettingsDraft(settings);
      showMessage(t("settings.invoiceEmailSaved"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setSavingInvoiceEmailSettings(false);
    }
  }, [applyInvoiceEmailSettingsDraft, invoiceEmailBodyTemplate, invoiceEmailReplyTo, invoiceEmailSubjectTemplate, showMessage, t]);

  const handleResetInvoiceEmailSettings = useCallback(async () => {
    setSavingInvoiceEmailSettings(true);
    try {
      const transport = await getTransport();
      const settings = await transport.saveInvoiceEmailSettings({
        subjectTemplate: "",
        bodyTemplate: "",
        replyTo: "",
      });
      applyInvoiceEmailSettingsDraft(settings);
      showMessage(t("settings.invoiceEmailResetDone"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setSavingInvoiceEmailSettings(false);
    }
  }, [applyInvoiceEmailSettingsDraft, showMessage, t]);

  useEffect(() => {
    if (!appReady || tab !== "settings" || !canManageSettings) return;
    void loadInvoiceEmailSettings();
  }, [appReady, canManageSettings, loadInvoiceEmailSettings, tab]);

  return {
    creatingBackup,
    invoiceArchive,
    invoiceArchiveLoading,
    invoiceEmailSettings,
    invoiceEmailSettingsLoading,
    savingInvoiceEmailSettings,
    invoiceEmailSubjectTemplate,
    invoiceEmailBodyTemplate,
    invoiceEmailReplyTo,
    users,
    usersLoading,
    creatingUser,
    newUserUsername,
    newUserPassword,
    newUserRole,
    userDrafts,
    userPasswordDrafts,
    setInvoiceEmailSubjectTemplate,
    setInvoiceEmailBodyTemplate,
    setInvoiceEmailReplyTo,
    setNewUserUsername,
    setNewUserPassword,
    setNewUserRole,
    setUserDrafts,
    setUserPasswordDrafts,
    createManualBackup,
    loadInvoiceArchive,
    loadUsers,
    handleCreateUser,
    handleSaveUser,
    handleDeleteUser,
    handleResetUserPassword,
    handleLocaleChange,
    handleSaveInvoiceEmailSettings,
    handleResetInvoiceEmailSettings,
  };
}
