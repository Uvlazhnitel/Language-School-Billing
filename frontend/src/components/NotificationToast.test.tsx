import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { NotificationToast } from "./NotificationToast";
import { createTranslator } from "../lib/i18n";

describe("NotificationToast", () => {
  it("renders as an alert for errors", () => {
    const markup = renderToStaticMarkup(
      <NotificationToast
        message={{ text: "Validation error", type: "error" }}
        onDismiss={() => {}}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain('role="alert"');
    expect(markup).toContain("Validation error");
    expect(markup).toContain("messageToast error");
    expect(markup).not.toContain("z-index");
    expect(markup).toContain("messageToastClose");
  });
});
