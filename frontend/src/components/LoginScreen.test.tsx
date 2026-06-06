import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { LoginScreen } from "./LoginScreen";
import { createTranslator } from "../lib/i18n";

describe("LoginScreen", () => {
  it("renders login fields and session-expired state", () => {
    const markup = renderToStaticMarkup(
      <LoginScreen
        username="admin"
        password=""
        rememberMe
        pending={false}
        error={null}
        sessionExpired
        onUsernameChange={vi.fn()}
        onPasswordChange={vi.fn()}
        onRememberMeChange={vi.fn()}
        onSubmit={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Sign in to StudentDesk");
    expect(markup).toContain("admin");
    expect(markup).toContain("Remember me");
    expect(markup).toContain("Your session expired");
  });
});
