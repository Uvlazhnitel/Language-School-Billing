import type { FormEvent } from "react";

import type { TranslateFn } from "../lib/i18n";

type LoginScreenProps = {
  email: string;
  password: string;
  pending: boolean;
  error: string | null;
  sessionExpired: boolean;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  t: TranslateFn;
};

export function LoginScreen(props: LoginScreenProps) {
  const {
    email,
    password,
    pending,
    error,
    sessionExpired,
    onEmailChange,
    onPasswordChange,
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
            <span>{t("auth.email")}</span>
            <input
              type="email"
              value={email}
              onChange={(event) => onEmailChange(event.target.value)}
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
