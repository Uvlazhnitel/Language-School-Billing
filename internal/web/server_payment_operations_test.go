package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"langschool/ent/auditlog"
	"langschool/internal/backend"
)

func TestPaymentOperationHTTPContract(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()
	ctx := context.Background()
	st := env.Runtime.DB.Ent.Student.Create().SetFullName("Payment operation").SaveX(ctx)
	payload := map[string]any{"studentId": st.ID, "amount": 10, "method": "cash", "paidAt": "2026-10-19", "note": "operation test"}
	for _, invalid := range []any{nil, "", "invalid", uuid.Nil.String()} {
		if invalid == nil {
			delete(payload, "operationId")
		} else {
			payload["operationId"] = invalid
		}
		encoded, _ := json.Marshal(payload)
		response, body := rawRequest(t, env.Client, http.MethodPost, env.Server.URL+"/api/payments", bytes.NewReader(encoded))
		if response.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "refresh the page") {
			t.Fatalf("invalid key response: %d %s", response.StatusCode, body)
		}
	}
	if env.Runtime.DB.Ent.Payment.Query().CountX(ctx) != 0 {
		t.Fatal("invalid request created payment")
	}
	payload["operationId"] = uuid.NewString()
	first := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", payload)
	second := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", payload)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("different response: %+v %+v", first, second)
	}
	if count := env.Runtime.DB.Ent.Payment.Query().CountX(ctx); count != 1 {
		t.Fatalf("payment count=%d", count)
	}
	if count := env.Runtime.DB.Ent.AuditLog.Query().Where(auditlog.ActionEQ("payment.allocate_or_credit")).CountX(ctx); count != 1 {
		t.Fatalf("audit count=%d", count)
	}
	payload["amount"] = 20
	encoded, _ := json.Marshal(payload)
	response, body := rawRequest(t, env.Client, http.MethodPost, env.Server.URL+"/api/payments", bytes.NewReader(encoded))
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("changed payload: %d %s", response.StatusCode, body)
	}
}
