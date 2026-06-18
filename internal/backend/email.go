package backend

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"langschool/ent/invoice"
	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
	auditsvc "langschool/internal/app/audit"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/app/recipient"
	"langschool/internal/email"
	appruntime "langschool/internal/runtime"
)

var invoiceEmailAvailablePlaceholders = []string{
	"{recipient_name}",
	"{invoice_number}",
	"{month_name}",
	"{year}",
	"{amount}",
	"{org_name}",
}

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

type invoiceEmailSettings struct {
	SubjectTemplate  string
	BodyTemplate     string
	ReplyTo          string
	OrganizationName string
}

func (s *Service) InvoiceEmailPreview(ctx context.Context, id int) (*InvoiceEmailPreviewResult, error) {
	dto, attachmentFilename, templateSettings, err := s.invoiceEmailDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	return &InvoiceEmailPreviewResult{
		To:                 strings.TrimSpace(dto.RecipientEmail),
		Subject:            renderInvoiceEmailTemplate(templateSettings.SubjectTemplate, dto, templateSettings.OrganizationName),
		Body:               renderInvoiceEmailTemplate(templateSettings.BodyTemplate, dto, templateSettings.OrganizationName),
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

	dto, _, templateSettings, err := s.invoiceEmailDraft(ctx, id)
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
		ReplyTo:            templateSettings.ReplyTo,
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

func (s *Service) SettingsGetInvoiceEmail(ctx context.Context) (*InvoiceEmailSettingsDTO, error) {
	templateSettings, err := s.invoiceEmailSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &InvoiceEmailSettingsDTO{
		SubjectTemplate:       templateSettings.SubjectTemplate,
		BodyTemplate:          templateSettings.BodyTemplate,
		ReplyTo:               templateSettings.ReplyTo,
		AvailablePlaceholders: append([]string(nil), invoiceEmailAvailablePlaceholders...),
	}, nil
}

func (s *Service) SettingsSetInvoiceEmail(ctx context.Context, subjectTemplate, bodyTemplate, replyTo string) (*InvoiceEmailSettingsDTO, error) {
	replyTo = strings.TrimSpace(replyTo)
	if replyTo != "" {
		if _, err := mail.ParseAddress(replyTo); err != nil {
			return nil, fmt.Errorf("invalid invoice Reply-To email")
		}
	}

	subjectTemplate = invoiceEmailTemplateOrDefault(subjectTemplate, appruntime.DefaultInvoiceEmailSubjectTemplate)
	bodyTemplate = invoiceEmailTemplateOrDefault(bodyTemplate, appruntime.DefaultInvoiceEmailBodyTemplate)

	_, err := s.rt.DB.Ent.Settings.
		Update().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		SetInvoiceEmailSubjectTemplate(subjectTemplate).
		SetInvoiceEmailBodyTemplate(bodyTemplate).
		SetInvoiceReplyTo(replyTo).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return s.SettingsGetInvoiceEmail(ctx)
}

func (s *Service) invoiceEmailDraft(ctx context.Context, id int) (*InvoiceDTO, string, invoiceEmailSettings, error) {
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return nil, "", invoiceEmailSettings{}, err
	}
	if dto.Number == nil || strings.TrimSpace(*dto.Number) == "" {
		return nil, "", invoiceEmailSettings{}, fmt.Errorf("счёт ещё не выставлен")
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
		return nil, "", invoiceEmailSettings{}, err
	}
	templateSettings, err := s.invoiceEmailSettings(ctx)
	if err != nil {
		return nil, "", invoiceEmailSettings{}, err
	}
	return dto, attachmentFilename, templateSettings, nil
}

func (s *Service) resolveInvoiceAttachmentFilename(ctx context.Context, dto *InvoiceDTO) (string, error) {
	iv, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.IDEQ(dto.ID)).
		WithStudent().
		Only(ctx)
	if err == nil {
		_, _, subjectName := archiveInvoiceNames(iv)
		info := s.invoicePDFInfo(iv, subjectName)
		if info.Status == invsvc.PDFStatusReady {
			return info.Filename, nil
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

func (s *Service) invoiceEmailSettings(ctx context.Context) (invoiceEmailSettings, error) {
	st, err := s.rt.DB.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		return invoiceEmailSettings{}, err
	}
	replyTo := strings.TrimSpace(st.InvoiceReplyTo)
	if replyTo != "" {
		if _, err := mail.ParseAddress(replyTo); err != nil {
			return invoiceEmailSettings{}, fmt.Errorf("invalid invoice Reply-To email")
		}
	}

	return invoiceEmailSettings{
		SubjectTemplate:  invoiceEmailTemplateOrDefault(st.InvoiceEmailSubjectTemplate, appruntime.DefaultInvoiceEmailSubjectTemplate),
		BodyTemplate:     invoiceEmailTemplateOrDefault(st.InvoiceEmailBodyTemplate, appruntime.DefaultInvoiceEmailBodyTemplate),
		ReplyTo:          replyTo,
		OrganizationName: s.organizationNameFromSettings(st.OrgName),
	}, nil
}

func (s *Service) organizationNameFromSettings(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return appruntime.DefaultSchoolDisplayName
	}
	return name
}

func renderInvoiceEmailTemplate(template string, dto *InvoiceDTO, orgName string) string {
	number := invoiceLabel(dto.Number, dto.ID)
	recipientName := strings.TrimSpace(dto.RecipientName)
	if recipientName == "" {
		recipientName = strings.TrimSpace(dto.StudentName)
	}
	replacements := map[string]string{
		"{recipient_name}": recipientName,
		"{invoice_number}": number,
		"{month_name}":     lvInvoiceMonthName(dto.Month),
		"{year}":           strconv.Itoa(dto.Year),
		"{amount}":         fmt.Sprintf("%.2f", dto.Total),
		"{org_name}":       strings.TrimSpace(orgName),
	}
	out := template
	for placeholder, value := range replacements {
		out = strings.ReplaceAll(out, placeholder, value)
	}
	return out
}

func invoiceEmailTemplateOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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
