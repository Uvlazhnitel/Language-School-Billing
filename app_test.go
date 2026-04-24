package main

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"langschool/ent/enttest"
	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
)

func TestDefaultSettingsCreateWithoutAutoIssue(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:appsettings?mode=memory&_fk=1")
	defer client.Close()

	st, err := client.Settings.Create().
		SetSingletonID(sharedapp.SettingsSingletonID).
		SetOrgName("").
		SetAddress("").
		SetInvoicePrefix("LS").
		SetNextSeq(1).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("lv-LV").
		Save(ctx)
	if err != nil {
		t.Fatalf("create settings without auto_issue: %v", err)
	}

	got, err := client.Settings.Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query settings: %v", err)
	}

	if got.ID != st.ID {
		t.Fatalf("got settings id %d, want %d", got.ID, st.ID)
	}
}
