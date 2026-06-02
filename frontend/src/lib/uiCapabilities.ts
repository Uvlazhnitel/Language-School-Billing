import type { TransportCapabilities } from "./api";

export function canShowInvoiceFolderAction(capabilities: TransportCapabilities): boolean {
  return capabilities.canOpenFolders;
}

export function canShowSettingsFilesCard(capabilities: TransportCapabilities): boolean {
  return capabilities.canOpenFolders;
}
