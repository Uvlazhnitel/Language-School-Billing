import { createContext } from "react";
import type { UiLocale } from "./i18n";
import type { TableSortPreferences } from "./tableSort";
export const SortContext = createContext<{ locale: UiLocale; preferences?: TableSortPreferences }>({
  locale: "lv-LV",
});
