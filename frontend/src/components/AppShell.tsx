type AppTab = {
  id: string;
  label: string;
  eyebrow: string;
  title: string;
  subtitle?: string;
};

type AppShellProps = {
  tabs: AppTab[];
  activeTab: string;
  secondaryActions?: Array<{ id: string; label: string; onClick: () => void }>;
  onTabChange: (tabId: string) => void;
  currentMeta: AppTab;
  month: number;
  year: number;
  monthLabels: string[];
  onMonthChange: (month: number) => void;
  onYearChange: (year: number) => void;
  showMonthPicker: boolean;
  children: React.ReactNode;
};

export function AppShell({
  tabs,
  activeTab,
  secondaryActions = [],
  onTabChange,
  currentMeta,
  month,
  year,
  monthLabels,
  onMonthChange,
  onYearChange,
  showMonthPicker,
  children,
}: AppShellProps) {
  return (
    <div className="appShell">
      <section className="workspaceCard workspaceCard--shell">
        <div className="workspaceTopbar workspaceTopbar--shell">
          <div className="workspaceHeading">
            <div className="workspaceEyebrow">{currentMeta.eyebrow}</div>
            <h1>{currentMeta.title}</h1>
            {currentMeta.subtitle && <p className="workspaceSubtitle">{currentMeta.subtitle}</p>}
          </div>

          {showMonthPicker && (
            <div className="monthpickers monthpickersTopbar">
              <select value={month} onChange={(e) => onMonthChange(parseInt(e.target.value, 10))}>
                {monthLabels.map((label, index) => (
                  <option key={label} value={index + 1}>
                    {label}
                  </option>
                ))}
              </select>
              <select value={year} onChange={(e) => onYearChange(parseInt(e.target.value, 10))}>
                {[year - 1, year, year + 1].map((value) => (
                  <option key={value} value={value}>
                    {value}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        <div className="workspaceNav">
          <nav className="tabs tabs--primary">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                className={activeTab === tab.id ? "active" : ""}
                onClick={() => onTabChange(tab.id)}
              >
                {tab.label}
              </button>
            ))}
          </nav>

          {secondaryActions.length > 0 && (
            <div className="secondaryActions">
              {secondaryActions.map((action) => (
                <button key={action.id} className="secondaryActionButton" onClick={action.onClick}>
                  {action.label}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="workspaceContent">{children}</div>
      </section>
    </div>
  );
}
