import { useCallback, useEffect, useState, type FormEvent } from "react";

import { getTransport } from "../../lib/api";
import { AUTH_REQUIRED_EVENT } from "../../lib/api/shared";
import { createTranslator, normalizeLocale, type UiLocale } from "../../lib/i18n";

type SessionUser = {
  id: number;
  username: string;
  role: string;
} | null;

type UseAuthControllerParams = {
  showMessage: (message: string, type?: "success" | "error") => void;
};

export function useAuthController({ showMessage }: UseAuthControllerParams) {
  const [appReady, setAppReady] = useState(false);
  const [authLoading, setAuthLoading] = useState(true);
  const [authRequired, setAuthRequired] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(true);
  const [currentSessionUser, setCurrentSessionUser] = useState<SessionUser>(null);
  const [sessionCapabilities, setSessionCapabilities] = useState<Record<string, boolean>>({});
  const [loginUsername, setLoginUsername] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [loginRememberMe, setLoginRememberMe] = useState(true);
  const [loginPending, setLoginPending] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [uiLocale, setUiLocale] = useState<UiLocale>("lv-LV");

  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
      try {
        const transport = await getTransport();
        const bootstrapResult = await transport.bootstrap();
        if (cancelled) return;
        setUiLocale(normalizeLocale(bootstrapResult.locale));
        setAuthRequired(bootstrapResult.authRequired);
        setIsAuthenticated(bootstrapResult.session.authenticated);
        setCurrentSessionUser(bootstrapResult.session.user ?? null);
        setSessionCapabilities(bootstrapResult.session.capabilities ?? {});
        setAppReady(
          bootstrapResult.ready &&
            (!bootstrapResult.authRequired || bootstrapResult.session.authenticated),
        );
        setAuthLoading(false);
      } catch (e: any) {
        if (!cancelled) {
          setAuthLoading(false);
          showMessage(
            createTranslator("lv-LV")("msg.loadingFoldersError", {
              message: String(e?.message ?? e),
            }),
            "error",
          );
        }
      }
    };

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, [showMessage]);

  useEffect(() => {
    const onAuthRequired = () => {
      setIsAuthenticated(false);
      setCurrentSessionUser(null);
      setSessionCapabilities({});
      setAppReady(false);
      setLoginPassword("");
      setLoginError(null);
      setSessionExpired(true);
    };

    window.addEventListener(AUTH_REQUIRED_EVENT, onAuthRequired);
    return () => {
      window.removeEventListener(AUTH_REQUIRED_EVENT, onAuthRequired);
    };
  }, []);

  const handleLogin = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      setLoginPending(true);
      setLoginError(null);
      try {
        const transport = await getTransport();
        const session = await transport.login(loginUsername, loginPassword, loginRememberMe);
        setUiLocale(normalizeLocale(session.locale));
        setCurrentSessionUser(session.user ?? null);
        setSessionCapabilities(session.capabilities ?? {});
        setIsAuthenticated(session.authenticated);
        setAppReady(session.ready && session.authenticated);
        setLoginError(null);
        setLoginPassword("");
        setSessionExpired(false);
      } catch (e: any) {
        setLoginError(String(e?.message ?? e));
      } finally {
        setLoginPending(false);
      }
    },
    [loginRememberMe, loginPassword, loginUsername],
  );

  const handleLogout = useCallback(async () => {
    try {
      const transport = await getTransport();
      await transport.logout();
    } catch (error) {
      void error;
    }
    setIsAuthenticated(false);
    setCurrentSessionUser(null);
    setSessionCapabilities({});
    setAppReady(false);
    setLoginPassword("");
    setLoginError(null);
    setSessionExpired(false);
  }, []);

  return {
    appReady,
    authLoading,
    authRequired,
    isAuthenticated,
    currentSessionUser,
    sessionCapabilities,
    loginUsername,
    loginPassword,
    loginRememberMe,
    loginPending,
    loginError,
    sessionExpired,
    uiLocale,
    setUiLocale,
    setLoginUsername,
    setLoginPassword,
    setLoginRememberMe,
    handleLogin,
    handleLogout,
  };
}
