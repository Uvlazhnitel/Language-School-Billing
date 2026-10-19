package payment

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"langschool/ent"
	"langschool/ent/paymentoperation"
	"langschool/internal/apperrors"
	"langschool/internal/money"
)

var ErrInvalidOperationID = errors.New("This payment form is outdated; refresh the page and try again")

// CreateWithOperation records one logical payment, including all its allocations.
// The returned boolean identifies a replay and lets callers avoid duplicate audit events.
func (s *Service) CreateWithOperation(ctx context.Context, actorID int, operationID string, studentID int, invoiceID *int, amount float64, method, paidAt, note string) (*PaymentDTO, bool, error) {
	id, err := uuid.Parse(operationID)
	if err != nil || id == uuid.Nil {
		return nil, false, ErrInvalidOperationID
	}
	operationID = id.String()
	date, err := parseDate(paidAt)
	if err != nil {
		return nil, false, fmt.Errorf("invalid payment date: %w", err)
	}
	payload, err := json.Marshal(struct {
		StudentID   int
		InvoiceID   *int
		AmountCents int64
		Method      string
		PaidAt      string
		Note        string
	}{studentID, invoiceID, money.EurosToCents(amount), method, date.UTC().Format(time.RFC3339Nano), note})
	if err != nil {
		return nil, false, err
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])

	// SQLite serializable starts BEGIN IMMEDIATE: reserve the writer before any
	// reads. The unique constraint arbitrates duplicates across processes too.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	operation, err := tx.PaymentOperation.Create().SetActorID(actorID).
		SetOperationID(operationID).SetRequestHash(hash).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if !ent.IsConstraintError(err) {
			return nil, false, err
		}
		existing, lookupErr := s.db.PaymentOperation.Query().Where(
			paymentoperation.ActorIDEQ(actorID), paymentoperation.OperationIDEQ(operationID),
		).Only(ctx)
		if lookupErr != nil {
			return nil, false, lookupErr
		}
		if existing.RequestHash != hash {
			return nil, false, apperrors.Conflict("payment operationId was already used with different parameters")
		}
		var result PaymentDTO
		if err := json.Unmarshal([]byte(existing.ResultJSON), &result); err != nil {
			return nil, false, fmt.Errorf("read saved payment operation: %w", err)
		}
		return &result, true, nil
	}

	result, err := (&Service{db: tx.Client()}).createInStore(ctx, studentID, invoiceID, amount, method, paidAt, note)
	if err != nil {
		return nil, false, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, false, err
	}
	if err := tx.PaymentOperation.UpdateOneID(operation.ID).SetResultJSON(string(encoded)).Exec(ctx); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return result, false, nil
}
