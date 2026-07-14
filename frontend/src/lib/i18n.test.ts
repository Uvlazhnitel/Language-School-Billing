import { describe, expect, it } from "vitest";

import { createTranslator, getMonthNames, normalizeLocale } from "./i18n";

describe("i18n", () => {
  it("defaults unknown locales to Latvian", () => {
    expect(normalizeLocale()).toBe("lv-LV");
    expect(normalizeLocale("unknown")).toBe("lv-LV");
    expect(normalizeLocale("lv-LV")).toBe("lv-LV");
  });

  it("returns Latvian month names", () => {
    expect(getMonthNames("lv-LV")[0]).toBe("Janvāris");
    expect(getMonthNames("lv-LV")[11]).toBe("Decembris");
  });

  it("translates Latvian interface strings", () => {
    const t = createTranslator("lv-LV");
    expect(t("tabs.students")).toBe("Skolēni");
    expect(t("settings.languageLatvian")).toBe("Latviešu");
    expect(t("msg.invoiceIssued", { number: "42" })).toBe("Rēķins izrakstīts: #42");
    expect(t("button.createAndOpenAttendance")).toBe("Izveidot un doties uz apmeklējumu");
  });

  it("translates onboarding actions in English and Russian", () => {
    expect(createTranslator("en-US")("button.enrollExistingStudent")).toBe(
      "Enroll existing student"
    );
    expect(createTranslator("ru-RU")("button.addToCourse")).toBe("Добавить в курс");
  });
});
