import type { FormEvent } from "react";

import type { TranslateFn } from "../lib/i18n";

type LoginScreenProps = {
  username: string;
  password: string;
  rememberMe: boolean;
  pending: boolean;
  error: string | null;
  sessionExpired: boolean;
  onUsernameChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onRememberMeChange: (value: boolean) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  t: TranslateFn;
};

export function LoginScreen(props: LoginScreenProps) {
  const {
    username,
    password,
    rememberMe,
    pending,
    error,
    sessionExpired,
    onUsernameChange,
    onPasswordChange,
    onRememberMeChange,
    onSubmit,
    t,
  } = props;

  return (
    <div className="authShell">
      <section className="authCard">
        <div className="workspaceEyebrow">{t("auth.eyebrow")}</div>
        <h1>{t("auth.title")}</h1>
        <p className="authCopy">{t("auth.subtitle")}</p>

        {sessionExpired && <div className="authNotice">{t("auth.sessionExpired")}</div>}
        {error && <div className="authError">{error}</div>}

        <form className="authForm" onSubmit={onSubmit}>
          <label className="authField">
            <span>{t("auth.username")}</span>
            <input
              type="text"
              value={username}
              onChange={(event) => onUsernameChange(event.target.value)}
              autoComplete="username"
              required
            />
          </label>

          <label className="authField">
            <span>{t("auth.password")}</span>
            <input
              type="password"
              value={password}
              onChange={(event) => onPasswordChange(event.target.value)}
              autoComplete="current-password"
              required
            />
          </label>

          <label className="authCheckbox">
            <input
              type="checkbox"
              checked={rememberMe}
              onChange={(event) => onRememberMeChange(event.target.checked)}
            />
            <span>{t("auth.rememberMe")}</span>
          </label>

          <button
            type="submit"
            className="workspaceActionButton workspaceActionButtonPrimary authSubmit"
            disabled={pending}
          >
            {pending ? `${t("auth.login")}...` : t("auth.login")}
          </button>
        </form>
      </section>
    </div>
  );
}
