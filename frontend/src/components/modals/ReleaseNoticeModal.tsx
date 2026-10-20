import { useEffect, useRef } from "react";
import type { ReleaseNoticeContent } from "../../lib/api/types";
import type { TranslateFn } from "../../lib/i18n";
type Props = { content: ReleaseNoticeContent; onClose: () => void; t: TranslateFn };
export function ReleaseNoticeModal({ content, onClose, t }: Props) {
  const ref = useRef<HTMLDialogElement>(null);
  const callback = useRef(onClose);
  callback.current = onClose;
  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    dialog.showModal();
    return () => {
      dialog.close();
      document.body.style.overflow = overflow;
      if (previous?.isConnected) previous.focus();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className="releaseNoticeModal"
      aria-modal="true"
      aria-labelledby="release-notice-title"
      onKeyDown={(event) => {
        if (event.key !== "Tab") return;
        const stops = [
          ...event.currentTarget.querySelectorAll<HTMLElement>(
            'button:not([disabled]), [tabindex="0"]'
          ),
        ];
        const first = stops[0];
        const last = stops[stops.length - 1];
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last?.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first?.focus();
        }
      }}
      onCancel={(event) => {
        event.preventDefault();
        callback.current();
      }}
    >
      <header className="releaseNoticeHeader">
        <span className="workspaceEyebrow">{t("releaseNotice.label")}</span>
        <button
          autoFocus
          type="button"
          className="releaseNoticeClose"
          aria-label={t("button.close")}
          onClick={onClose}
        >
          ×
        </button>
      </header>
      <div
        className="releaseNoticeScroll"
        tabIndex={0}
        role="region"
        aria-label={t("releaseNotice.label")}
      >
        <article className="releaseNoticeContent">
          <div className="releaseNoticeHero" aria-hidden="true">
            ✦
          </div>
          <h1 id="release-notice-title">{content.title}</h1>
          <p className="releaseNoticeIntro">{content.intro}</p>
          <div className="releaseNoticeSections">
            {content.sections.map((section, index) => (
              <section key={section.title}>
                <span className="releaseNoticeNumber" aria-hidden="true">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <div>
                  <h2>{section.title}</h2>
                  <p>{section.text}</p>
                </div>
              </section>
            ))}
          </div>
          <p className="releaseNoticeHint">{t("releaseNotice.hint")}</p>
        </article>
      </div>
      <footer className="releaseNoticeFooter">
        <button className="btn btnPrimary" type="button" onClick={onClose}>
          {t("releaseNotice.understood")}
        </button>
      </footer>
    </dialog>
  );
}
