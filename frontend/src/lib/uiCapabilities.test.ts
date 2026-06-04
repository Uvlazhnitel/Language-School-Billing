import { describe, expect, it } from "vitest";

import { canShowInvoiceFolderAction, canShowSettingsFilesCard } from "./uiCapabilities";

describe("ui capabilities", () => {
  it("shows folder actions in desktop mode", () => {
    const capabilities = {
      isDesktop: true,
      canOpenLocalFiles: true,
      canOpenFolders: true,
      canDownloadPdf: true,
    };

    expect(canShowInvoiceFolderAction(capabilities)).toBe(true);
    expect(canShowSettingsFilesCard(capabilities)).toBe(true);
  });

  it("hides folder actions in web mode", () => {
    const capabilities = {
      isDesktop: false,
      canOpenLocalFiles: false,
      canOpenFolders: false,
      canDownloadPdf: true,
    };

    expect(canShowInvoiceFolderAction(capabilities)).toBe(false);
    expect(canShowSettingsFilesCard(capabilities)).toBe(false);
  });
});
