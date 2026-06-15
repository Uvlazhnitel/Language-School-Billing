package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
	auditsvc "langschool/internal/app/audit"
	"langschool/internal/app/recipient"
	"langschool/internal/email"
	appruntime "langschool/internal/runtime"
)

type InvoiceEmailPreviewResult struct {
	To                 string `json:"to"`
	Subject            string `json:"subject"`
	Body               string `json:"body"`
	AttachmentFilename string `json:"attachmentFilename"`
}

type InvoiceSendEmailResult struct {
	To                 string `json:"to"`
	Subject            string `json:"subject"`
	AttachmentFilename string `json:"attachmentFilename"`
	SentAt             string `json:"sentAt"`
}

func (s *Service) InvoiceEmailPreview(ctx context.Context, id int) (*InvoiceEmailPreviewResult, error) {
	dto, attachmentFilename, orgName, err := s.invoiceEmailDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	return &InvoiceEmailPreviewResult{
		To:                 strings.TrimSpace(dto.RecipientEmail),
		Subject:            buildInvoiceEmailSubject(dto),
		Body:               buildInvoiceEmailBody(dto, orgName),
		AttachmentFilename: attachmentFilename,
	}, nil
}

func (s *Service) InvoiceSendEmail(ctx context.Context, id int, to, subject, body string) (*InvoiceSendEmailResult, error) {
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if to == "" {
		return nil, fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return nil, fmt.Errorf("email subject is required")
	}
	if body == "" {
		return nil, fmt.Errorf("email body is required")
	}

	dto, _, _, err := s.invoiceEmailDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	pdfPath, err := s.InvoiceEnsurePDF(ctx, id)
	if err != nil {
		return nil, err
	}
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read invoice pdf: %w", err)
	}
	if s.emailSender == nil {
		return nil, fmt.Errorf(email.ErrNotConfiguredText)
	}
	filename := filepath.Base(pdfPath)
	if err := s.emailSender.Send(ctx, email.Message{
		To:                 to,
		Subject:            subject,
		Body:               body,
		AttachmentFilename: filename,
		AttachmentData:     pdfData,
	}); err != nil {
		return nil, err
	}

	sentAt := time.Now().UTC().Format(time.RFC3339)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		InvoiceID:  intPtr(id),
		StudentID:  intPtr(dto.StudentID),
		Action:     "invoice.send_email",
		Summary:    fmt.Sprintf("Sent invoice %s to %s", invoiceLabel(dto.Number, dto.ID), to),
		After: map[string]any{
			"to":                 to,
			"subject":            subject,
			"attachmentFilename": filename,
			"sentAt":             sentAt,
		},
	})

	return &InvoiceSendEmailResult{
		To:                 to,
		Subject:            subject,
		AttachmentFilename: filename,
		SentAt:             sentAt,
	}, nil
}

func (s *Service) invoiceEmailDraft(ctx context.Context, id int) (*InvoiceDTO, string, string, error) {
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return nil, "", "", err
	}
	if dto.Number == nil || strings.TrimSpace(*dto.Number) == "" {
		return nil, "", "", fmt.Errorf("счёт ещё не выставлен")
	}
	recipientInfo, err := recipient.ResolveInvoiceRecipient(ctx, s.rt.DB.Ent, dto.StudentID)
	if err == nil {
		dto.RecipientName = recipientInfo.RecipientName
		dto.RecipientPhone = recipientInfo.RecipientPhone
		dto.RecipientEmail = recipientInfo.RecipientEmail
		dto.ChildName = recipientInfo.ChildName
		dto.StudentPersonalCode = recipientInfo.StudentPersonalCode
		dto.IsMinor = recipientInfo.IsMinor
	}

	attachmentFilename, err := s.resolveInvoiceAttachmentFilename(ctx, dto)
	if err != nil {
		return nil, "", "", err
	}
	orgName, err := s.organizationName(ctx)
	if err != nil {
		return nil, "", "", err
	}
	return dto, attachmentFilename, orgName, nil
}

func (s *Service) resolveInvoiceAttachmentFilename(ctx context.Context, dto *InvoiceDTO) (string, error) {
	for _, path := range s.invoicePDFPaths(dto) {
		if _, err := os.Stat(path); err == nil {
			return filepath.Base(path), nil
		}
	}
	fonts, err := appruntime.ResolveFontsDir(s.rt.Config, s.rt.Dirs)
	if err != nil {
		return "", err
	}
	_, pdfPath, err := s.rt.Invoice.Issue(ctx, dto.ID, s.rt.Dirs.Invoices, fonts)
	if err != nil {
		return "", err
	}
	return filepath.Base(pdfPath), nil
}

func (s *Service) organizationName(ctx context.Context) (string, error) {
	st, err := s.rt.DB.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(st.OrgName)
	if name == "" {
		name = appruntime.DefaultSchoolDisplayName
	}
	return name, nil
}

func buildInvoiceEmailSubject(dto *InvoiceDTO) string {
	number := invoiceLabel(dto.Number, dto.ID)
	return fmt.Sprintf("Rēķins %s par %s %d", number, lvInvoiceMonthName(dto.Month), dto.Year)
}

func buildInvoiceEmailBody(dto *InvoiceDTO, orgName string) string {
	number := invoiceLabel(dto.Number, dto.ID)
	recipientName := strings.TrimSpace(dto.RecipientName)
	if recipientName == "" {
		recipientName = strings.TrimSpace(dto.StudentName)
	}
	return fmt.Sprintf(
		"Labdien, %s!\n\nPielikumā nosūtām rēķinu %s par %s %d summā %.2f EUR.\n\nJa ir jautājumi, lūdzu, sazinieties ar mums.\n\nAr cieņu,\n%s",
		recipientName,
		number,
		lvInvoiceMonthName(dto.Month),
		dto.Year,
		dto.Total,
		orgName,
	)
}

func lvInvoiceMonthName(month int) string {
	names := []string{
		"janvāri",
		"februāri",
		"martu",
		"aprīli",
		"maiju",
		"jūniju",
		"jūliju",
		"augustu",
		"septembri",
		"oktobri",
		"novembri",
		"decembri",
	}
	if month < 1 || month > len(names) {
		return fmt.Sprintf("%02d", month)
	}
	return names[month-1]
}
