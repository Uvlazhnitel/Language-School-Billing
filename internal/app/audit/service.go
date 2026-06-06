package audit

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/auditlog"
)

type RecordEvent struct {
	ActorUserID *int
	ActorLabel  string
	EntityType  string
	EntityID    *int
	Action      string
	Summary     string
	Before      any
	After       any
	StudentID   *int
	InvoiceID   *int
}

type ListFilter struct {
	Query      string
	ActorLabel string
	EntityType string
	Action     string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

type ListItem struct {
	ID          int    `json:"id"`
	ActorUserID *int   `json:"actorUserId,omitempty"`
	ActorLabel  string `json:"actorLabel"`
	EntityType  string `json:"entityType"`
	EntityID    *int   `json:"entityId,omitempty"`
	Action      string `json:"action"`
	Summary     string `json:"summary"`
	BeforeJSON  string `json:"beforeJson"`
	AfterJSON   string `json:"afterJson"`
	StudentID   *int   `json:"studentId,omitempty"`
	InvoiceID   *int   `json:"invoiceId,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type ListResult struct {
	Items    []ListItem `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

type Service struct{ db *ent.Client }

func New(db *ent.Client) *Service { return &Service{db: db} }

func (s *Service) Record(ctx context.Context, event RecordEvent) error {
	if s == nil || s.db == nil {
		return nil
	}

	beforeJSON, err := marshalSnapshot(event.Before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalSnapshot(event.After)
	if err != nil {
		return err
	}

	actorLabel := strings.TrimSpace(event.ActorLabel)
	if actorLabel == "" {
		actorLabel = "system"
	}

	builder := s.db.AuditLog.Create().
		SetActorLabel(actorLabel).
		SetEntityType(strings.TrimSpace(event.EntityType)).
		SetAction(strings.TrimSpace(event.Action)).
		SetSummary(strings.TrimSpace(event.Summary)).
		SetBeforeJSON(beforeJSON).
		SetAfterJSON(afterJSON)
	if event.ActorUserID != nil {
		builder.SetActorUserID(*event.ActorUserID)
	}
	if event.EntityID != nil {
		builder.SetEntityID(*event.EntityID)
	}
	if event.StudentID != nil {
		builder.SetStudentID(*event.StudentID)
	}
	if event.InvoiceID != nil {
		builder.SetInvoiceID(*event.InvoiceID)
	}
	_, err = builder.Save(ctx)
	return err
}

func (s *Service) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.AuditLog.Query()
	query = applyFilter(query, filter)

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}

	items, err := query.
		Order(ent.Desc(auditlog.FieldCreatedAt), ent.Desc(auditlog.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ListItem, 0, len(items))
	for _, item := range items {
		out = append(out, toListItem(item))
	}

	return &ListResult{
		Items:    out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func applyFilter(query *ent.AuditLogQuery, filter ListFilter) *ent.AuditLogQuery {
	queryText := strings.TrimSpace(filter.Query)
	if queryText != "" {
		query = query.Where(
			auditlog.Or(
				auditlog.SummaryContainsFold(queryText),
				auditlog.ActorLabelContainsFold(queryText),
				auditlog.ActionContainsFold(queryText),
				auditlog.EntityTypeContainsFold(queryText),
			),
		)
	}
	if actor := strings.TrimSpace(filter.ActorLabel); actor != "" {
		query = query.Where(auditlog.ActorLabelEqualFold(actor))
	}
	if entityType := strings.TrimSpace(filter.EntityType); entityType != "" {
		query = query.Where(auditlog.EntityTypeEqualFold(entityType))
	}
	if action := strings.TrimSpace(filter.Action); action != "" {
		query = query.Where(auditlog.ActionEqualFold(action))
	}
	if from := parseDateBoundary(filter.DateFrom, false); !from.IsZero() {
		query = query.Where(auditlog.CreatedAtGTE(from))
	}
	if to := parseDateBoundary(filter.DateTo, true); !to.IsZero() {
		query = query.Where(auditlog.CreatedAtLTE(to))
	}
	return query
}

func parseDateBoundary(value string, endOfDay bool) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	if endOfDay {
		return t.Add(24*time.Hour - time.Nanosecond)
	}
	return t
}

func marshalSnapshot(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func toListItem(item *ent.AuditLog) ListItem {
	return ListItem{
		ID:          item.ID,
		ActorUserID: item.ActorUserID,
		ActorLabel:  item.ActorLabel,
		EntityType:  item.EntityType,
		EntityID:    item.EntityID,
		Action:      item.Action,
		Summary:     item.Summary,
		BeforeJSON:  item.BeforeJSON,
		AfterJSON:   item.AfterJSON,
		StudentID:   item.StudentID,
		InvoiceID:   item.InvoiceID,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
	}
}
