import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { LoginScreen } from "./LoginScreen";
import { createTranslator } from "../lib/i18n";

describe("LoginScreen", () => {
  it("renders login fields and session-expired state", () => {
    const markup = renderToStaticMarkup(
      <LoginScreen
        email="admin@example.com"
        password=""
        pending={false}
        error={null}
        sessionExpired
        onEmailChange={vi.fn()}
        onPasswordChange={vi.fn()}
        onSubmit={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Sign in to StudentDesk");
    expect(markup).toContain("admin@example.com");
    expect(markup).toContain("Your session expired");
  });
});
