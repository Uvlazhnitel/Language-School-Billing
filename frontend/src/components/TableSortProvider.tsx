import { useState, type ReactNode } from "react";
import { TableSortPreferences } from "../lib/tableSort";
import { SortContext } from "../lib/tableSortContext";
import type { UiLocale } from "../lib/i18n";

export function TableSortProvider({
  accountId,
  locale,
  children,
}: {
  accountId?: number;
  locale: UiLocale;
  children: ReactNode;
}) {
  const [preferences] = useState(
    () =>
      new TableSortPreferences(String(accountId ?? "anonymous"), () =>
        typeof window === "undefined" ? undefined : window.localStorage
      )
  );
  return <SortContext.Provider value={{ locale, preferences }}>{children}</SortContext.Provider>;
}
