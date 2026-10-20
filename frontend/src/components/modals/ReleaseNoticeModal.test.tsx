import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ReleaseNoticeModal } from "./ReleaseNoticeModal";
import { AppShell } from "../AppShell";
import { createTranslator } from "../../lib/i18n";
describe("release notice interface", () => {
  it.each(["ru-RU", "en-US", "lv-LV"] as const)(
    "renders a named accessible dialog and translated close actions: %s",
    (locale) => {
      const t = createTranslator(locale);
      const html = renderToStaticMarkup(
        <ReleaseNoticeModal
          content={{
            title: "New <features>",
            intro: "Intro",
            sections: [{ title: "Courses", text: "Details" }],
          }}
          onClose={vi.fn()}
          t={t}
        />
      );
      expect(html).toContain('aria-modal="true"');
      expect(html).toContain('aria-labelledby="release-notice-title"');
      expect(html).toContain("New &lt;features&gt;");
      expect(html).toContain(t("releaseNotice.understood"));
      expect(html).toContain('aria-label="' + t("button.close") + '"');
    }
  );
  it("provides the trigger in AppShell without relying on the selected section", () => {
    const meta = {
      id: "attendance",
      label: "Attendance",
      title: "Attendance",
      eyebrow: "Tracking",
    };
    const html = renderToStaticMarkup(
      <AppShell
        tabs={[meta]}
        activeTab="attendance"
        currentMeta={meta}
        month={10}
        year={2026}
        monthLabels={["October"]}
        onMonthChange={vi.fn()}
        onYearChange={vi.fn()}
        onTabChange={vi.fn()}
        showMonthPicker={false}
        releaseNoticeAvailable
        releaseNoticeLabel="What’s new"
        onOpenReleaseNotice={vi.fn()}
      >
        <p>Page</p>
      </AppShell>
    );
    expect(html).toContain('aria-haspopup="dialog"');
    expect(html).toContain("What’s new");
  });
});
