import { AppDirs, OpenFile as OpenFileFunction } from "../../wailsjs/go/main/App";

export async function appDirs(): Promise<{
  base: string; data: string; backups: string; invoices: string; exports: string;
}> {
  return (await AppDirs()) as any;
}

export async function openFile(path: string): Promise<void> {
  await OpenFileFunction(path);
}
