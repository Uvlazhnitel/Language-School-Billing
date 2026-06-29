package backend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/invoice"
	sharedapp "langschool/internal/app"
	auditsvc "langschool/internal/app/audit"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/money"
	appruntime "langschool/internal/runtime"
)

func (s *Service) InvoiceGenerateDrafts(ctx context.Context, year, month int) (invsvc.GenerateResult, error) {
	before, _ := s.auditSnapshotForPeriod(ctx, year, month)
	result, err := s.rt.Invoice.GenerateDrafts(ctx, year, month)
	if err != nil {
		return result, err
	}
	after, _ := s.auditSnapshotForPeriod(ctx, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.generate_drafts",
		Summary: fmt.Sprintf(
			"Generated drafts for %04d-%02d: created %d, updated %d, skipped with invoices %d, skipped empty %d",
			year, month, result.Created, result.Updated, result.SkippedHasInvoice, result.SkippedNoLines,
		),
		Before: before,
		After:  after,
	})
	return result, nil
}

func (s *Service) InvoiceRebuildStudentDraft(ctx context.Context, studentID, year, month int) (invsvc.GenerateResult, error) {
	before, _ := s.auditSnapshotForStudentMonth(ctx, studentID, year, month)
	result, err := s.rt.Invoice.RebuildStudentDraft(ctx, studentID, year, month)
	if err != nil {
		return result, err
	}
	after, _ := s.auditSnapshotForStudentMonth(ctx, studentID, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.rebuild_student_draft",
		Summary: fmt.Sprintf(
			"Rebuilt draft invoices for student %d in %04d-%02d: created %d, updated %d, skipped with invoices %d, skipped empty %d",
			studentID, year, month, result.Created, result.Updated, result.SkippedHasInvoice, result.SkippedNoLines,
		),
		Before:    before,
		After:     after,
		StudentID: intPtr(studentID),
	})
	return result, nil
}

func (s *Service) InvoiceGet(ctx context.Context, id int) (*InvoiceDTO, error) {
	return s.rt.Invoice.Get(ctx, id)
}

func (s *Service) InvoiceDeleteDraft(ctx context.Context, id int) error {
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.DeleteDraft(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.delete_draft",
		Summary:    fmt.Sprintf("Deleted draft invoice for %s, %04d-%02d", meta.StudentName, meta.Year, meta.Month),
		Before:     before,
		After: map[string]any{
			"deleted":   true,
			"invoiceId": id,
		},
		StudentID: meta.StudentID,
		InvoiceID: intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceDeleteDraftWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.DeleteDraftWithVersion(ctx, id, version); err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.delete_draft",
		Summary:    fmt.Sprintf("Deleted draft invoice for %s, %04d-%02d", meta.StudentName, meta.Year, meta.Month),
		Before:     before,
		After: map[string]any{
			"deleted":   true,
			"invoiceId": id,
		},
		StudentID: meta.StudentID,
		InvoiceID: intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceReopenDraft(ctx context.Context, id int) error {
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.ReopenDraft(ctx, id, s.rt.Dirs.Invoices); err != nil {
		return err
	}
	after, _, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.reopen_draft",
		Summary:    fmt.Sprintf("Returned invoice %s to draft", invoiceLabel(meta.Number, id)),
		Before:     before,
		After:      after,
		StudentID:  meta.StudentID,
		InvoiceID:  intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceReopenDraftWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.ReopenDraftWithVersion(ctx, id, version, s.rt.Dirs.Invoices); err != nil {
		return err
	}
	after, _, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.reopen_draft",
		Summary:    fmt.Sprintf("Returned invoice %s to draft", invoiceLabel(meta.Number, id)),
		Before:     before,
		After:      after,
		StudentID:  meta.StudentID,
		InvoiceID:  intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceList(ctx context.Context, year, month int, status string) ([]invsvc.ListItem, error) {
	return s.rt.Invoice.List(ctx, year, month, status)
}

func (s *Service) InvoiceIssue(ctx context.Context, id int) (IssueResult, error) {
	before, meta, err := s.auditStudentFinanceSnapshotByInvoice(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	num, err := s.rt.Invoice.IssueOne(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	if err := s.rt.Payment.ApplyCreditToOldestInvoices(ctx, dto.StudentID); err != nil {
		return IssueResult{}, err
	}
	pdfReady, err := s.ensureIssuedInvoicePDF(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, dto.StudentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "invoice",
			EntityID:   intPtr(id),
			Action:     "invoice.issue",
			Summary:    fmt.Sprintf("Issued invoice %s for %s, total %.2f", num, meta.StudentName, dto.Total),
			Before:     before,
			After:      after,
			StudentID:  intPtr(dto.StudentID),
			InvoiceID:  intPtr(id),
		})
	}
	return IssueResult{Number: num, PDFReady: pdfReady, PDFStatus: issuePDFStatus(pdfReady)}, nil
}

func (s *Service) InvoiceIssueWithVersion(ctx context.Context, id, version int) (IssueResult, error) {
	if err := validateVersion(version); err != nil {
		return IssueResult{}, err
	}
	before, meta, err := s.auditStudentFinanceSnapshotByInvoice(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	num, studentID, err := s.rt.Invoice.IssueAndApplyCreditWithVersion(ctx, id, version)
	if err != nil {
		return IssueResult{}, err
	}
	pdfReady, err := s.ensureIssuedInvoicePDF(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "invoice",
			EntityID:   intPtr(id),
			Action:     "invoice.issue",
			Summary:    fmt.Sprintf("Issued invoice %s for %s, total %.2f", num, meta.StudentName, dto.Total),
			Before:     before,
			After:      after,
			StudentID:  intPtr(studentID),
			InvoiceID:  intPtr(id),
		})
	}
	return IssueResult{Number: num, PDFReady: pdfReady, PDFStatus: issuePDFStatus(pdfReady)}, nil
}

func (s *Service) InvoiceIssueAll(ctx context.Context, year, month int) (IssueAllResult, error) {
	before, _ := s.auditSnapshotForPeriod(ctx, year, month)
	drafts, err := s.rt.DB.Ent.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(year),
			invoice.PeriodMonthEQ(month),
			invoice.StatusEQ(invoice.Status(sharedapp.InvoiceStatusDraft)),
		).
		Order(ent.Asc(invoice.FieldID)).
		All(ctx)
	if err != nil {
		return IssueAllResult{}, err
	}

	result := IssueAllResult{
		PdfPaths: make([]string, 0, len(drafts)),
	}
	for _, draft := range drafts {
		if _, err := s.rt.Invoice.IssueOne(ctx, draft.ID); err != nil {
			return IssueAllResult{}, err
		}
		result.Count++
		ready, path, err := s.ensureIssuedInvoicePDFWithPath(ctx, draft.ID)
		if err != nil {
			return IssueAllResult{}, err
		}
		if ready {
			result.GeneratedCount++
			if path != "" {
				result.PdfPaths = append(result.PdfPaths, path)
			}
		} else {
			result.PendingCount++
		}
	}
	items, err := s.rt.Invoice.List(ctx, year, month, "all")
	if err != nil {
		return IssueAllResult{}, err
	}
	seen := make(map[int]struct{})
	for _, item := range items {
		if item.Status == sharedapp.InvoiceStatusDraft {
			continue
		}
		if _, ok := seen[item.StudentID]; ok {
			continue
		}
		seen[item.StudentID] = struct{}{}
		if err := s.rt.Payment.ApplyCreditToOldestInvoices(ctx, item.StudentID); err != nil {
			return IssueAllResult{}, err
		}
	}
	after, _ := s.auditSnapshotForPeriod(ctx, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.issue_all",
		Summary:    fmt.Sprintf("Issued %d invoices for %04d-%02d", result.Count, year, month),
		Before:     before,
		After:      after,
	})
	return result, nil
}

func (s *Service) ensureIssuedInvoicePDF(ctx context.Context, id int) (bool, error) {
	ready, _, err := s.ensureIssuedInvoicePDFWithPath(ctx, id)
	return ready, err
}

func (s *Service) ensureIssuedInvoicePDFWithPath(ctx context.Context, id int) (bool, string, error) {
	path, err := s.InvoiceEnsurePDF(ctx, id)
	if err != nil {
		log.Printf("Invoice ensure PDF fallback for invoice %d failed: %v", id, err)
	}
	ready, readyErr := s.InvoiceHasPDF(ctx, id)
	if readyErr != nil {
		return false, "", readyErr
	}
	if !ready {
		path = ""
	}
	return ready, path, nil
}

func (s *Service) InvoiceEnsurePDF(ctx context.Context, id int) (string, error) {
	iv, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.IDEQ(id)).
		WithStudent().
		Only(ctx)
	if err != nil {
		return "", err
	}
	if iv.Number == nil || strings.TrimSpace(*iv.Number) == "" {
		return "", fmt.Errorf("счёт ещё не выставлен")
	}
	_, _, subjectName := archiveInvoiceNames(iv)
	info := s.invoicePDFInfo(iv, subjectName)
	if info.Status == invsvc.PDFStatusReady && info.Path != "" {
		if sharedapp.InvoiceStatusIsPendingPDF(string(iv.Status)) {
			if err := s.rt.Payment.RecomputeInvoiceStatus(ctx, iv.ID); err != nil {
				return "", err
			}
		}
		return info.Path, nil
	}
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if dto.Number == nil || *dto.Number == "" {
		return "", fmt.Errorf("счёт ещё не выставлен")
	}
	fonts, err := appruntime.ResolveFontsDir(s.rt.Config, s.rt.Dirs)
	if err != nil {
		return "", err
	}
	_, path, err := s.rt.Invoice.Issue(ctx, id, s.rt.Dirs.Invoices, fonts)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) InvoiceEnsurePDFAll(ctx context.Context, year, month int) (EnsureAllPDFsResult, error) {
	result := EnsureAllPDFsResult{
		Year:  year,
		Month: month,
		Items: make([]EnsureAllPDFsItemResult, 0),
	}
	before, _ := s.auditSnapshotForPeriod(ctx, year, month)

	candidates, err := s.rt.DB.Ent.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(year),
			invoice.PeriodMonthEQ(month),
			invoice.StatusIn(
				invoice.Status(sharedapp.InvoiceStatusIssuedPendingPDF),
				invoice.Status(sharedapp.InvoiceStatusPaidPendingPDF),
			),
		).
		WithStudent().
		Order(ent.Asc(invoice.FieldID)).
		All(ctx)
	if err != nil {
		return EnsureAllPDFsResult{}, err
	}

	result.Items = make([]EnsureAllPDFsItemResult, 0, len(candidates))
	for _, candidate := range candidates {
		fresh, err := s.rt.DB.Ent.Invoice.Query().
			Where(invoice.IDEQ(candidate.ID)).
			WithStudent().
			Only(ctx)
		if err != nil {
			result.Items = append(result.Items, EnsureAllPDFsItemResult{
				InvoiceID:   candidate.ID,
				Number:      strings.TrimSpace(derefString(candidate.Number)),
				StudentName: strings.TrimSpace(candidate.Edges.Student.FullName),
				Status:      string(candidate.Status),
				Result:      "failed",
				Message:     err.Error(),
			})
			result.FailedCount++
			continue
		}

		studentName, _, subjectName := archiveInvoiceNames(fresh)
		info := s.invoicePDFInfo(fresh, subjectName)
		item := EnsureAllPDFsItemResult{
			InvoiceID:   fresh.ID,
			Number:      strings.TrimSpace(derefString(fresh.Number)),
			StudentName: studentName,
			Status:      string(fresh.Status),
		}

		if info.Status == invsvc.PDFStatusReady {
			if sharedapp.InvoiceStatusIsPendingPDF(string(fresh.Status)) {
				if err := s.rt.Payment.RecomputeInvoiceStatus(ctx, fresh.ID); err != nil {
					item.Result = "failed"
					item.Message = err.Error()
					result.FailedCount++
					result.Items = append(result.Items, item)
					continue
				}
				refreshed, err := s.rt.DB.Ent.Invoice.Query().
					Where(invoice.IDEQ(fresh.ID)).
					WithStudent().
					Only(ctx)
				if err == nil {
					item.Status = string(refreshed.Status)
				}
			}
			item.Result = "already_ready"
			result.AlreadyReadyCount++
			result.Items = append(result.Items, item)
			continue
		}

		if _, err := s.InvoiceEnsurePDF(ctx, fresh.ID); err != nil {
			item.Result = "failed"
			item.Message = err.Error()
			result.FailedCount++
			result.Items = append(result.Items, item)
			continue
		}

		updated, err := s.rt.DB.Ent.Invoice.Query().
			Where(invoice.IDEQ(fresh.ID)).
			WithStudent().
			Only(ctx)
		if err != nil {
			item.Result = "failed"
			item.Message = err.Error()
			result.FailedCount++
			result.Items = append(result.Items, item)
			continue
		}
		item.Status = string(updated.Status)
		item.Result = "generated"
		result.GeneratedCount++
		result.Items = append(result.Items, item)
	}

	result.Processed = len(result.Items)
	after, _ := s.auditSnapshotForPeriod(ctx, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.ensure_pdf_all",
		Summary: fmt.Sprintf(
			"Ensured PDFs for %04d-%02d: generated %d, already ready %d, failed %d",
			year,
			month,
			result.GeneratedCount,
			result.AlreadyReadyCount,
			result.FailedCount,
		),
		Before: before,
		After:  after,
	})
	return result, nil
}

func (s *Service) InvoiceHasPDF(ctx context.Context, id int) (bool, error) {
	iv, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.IDEQ(id)).
		WithStudent().
		Only(ctx)
	if err != nil {
		return false, err
	}
	_, _, subjectName := archiveInvoiceNames(iv)
	info := s.invoicePDFInfo(iv, subjectName)
	return info.Status == invsvc.PDFStatusReady, nil
}

func (s *Service) InvoiceArchiveList(ctx context.Context) (*InvoiceArchiveResult, error) {
	items, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StatusNEQ(invoice.StatusDraft)).
		WithStudent().
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &InvoiceArchiveResult{Years: []InvoiceArchiveYearDTO{}}, nil
	}

	archiveItems := make([]InvoiceArchiveInvoiceDTO, 0, len(items))
	for _, item := range items {
		archiveItems = append(archiveItems, s.invoiceArchiveInvoice(item))
	}

	sort.Slice(archiveItems, func(i, j int) bool {
		if archiveItems[i].Year != archiveItems[j].Year {
			return archiveItems[i].Year > archiveItems[j].Year
		}
		if archiveItems[i].Month != archiveItems[j].Month {
			return archiveItems[i].Month > archiveItems[j].Month
		}
		return archiveItems[i].InvoiceID > archiveItems[j].InvoiceID
	})

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	yearMap := make(map[int][]InvoiceArchiveInvoiceDTO)
	for _, item := range archiveItems {
		yearMap[item.Year] = append(yearMap[item.Year], item)
	}

	years := make([]InvoiceArchiveYearDTO, 0, len(yearMap))
	for year, yearItems := range yearMap {
		monthMap := make(map[int][]InvoiceArchiveInvoiceDTO)
		for _, item := range yearItems {
			monthMap[item.Month] = append(monthMap[item.Month], item)
		}

		monthKeys := make([]int, 0, len(monthMap))
		for month := range monthMap {
			monthKeys = append(monthKeys, month)
		}
		sort.Slice(monthKeys, func(i, j int) bool {
			left, right := monthKeys[i], monthKeys[j]
			if year == currentYear {
				if left == currentMonth && right != currentMonth {
					return true
				}
				if right == currentMonth && left != currentMonth {
					return false
				}
			}
			return left > right
		})

		months := make([]InvoiceArchiveMonthDTO, 0, len(monthKeys))
		for _, month := range monthKeys {
			invoices := monthMap[month]
			sort.Slice(invoices, func(i, j int) bool {
				return invoices[i].InvoiceID > invoices[j].InvoiceID
			})
			readyPDFCount := 0
			missingPDFCount := 0
			for _, archived := range invoices {
				if archived.PDFStatus == invoiceArchivePDFStatusReady {
					readyPDFCount++
				} else {
					missingPDFCount++
				}
			}
			zipDownloadURL := ""
			if readyPDFCount > 0 {
				zipDownloadURL = invoiceArchiveMonthZipURL(year, month)
			}
			months = append(months, InvoiceArchiveMonthDTO{
				Month:             month,
				Count:             len(invoices),
				ReadyPDFCount:     readyPDFCount,
				MissingPDFCount:   missingPDFCount,
				ZipDownloadURL:    zipDownloadURL,
				ExpandedByDefault: year == currentYear && month == currentMonth,
				Invoices:          invoices,
			})
		}

		years = append(years, InvoiceArchiveYearDTO{
			Year:              year,
			Count:             len(yearItems),
			ExpandedByDefault: year == currentYear,
			Months:            months,
		})
	}

	sort.Slice(years, func(i, j int) bool {
		return years[i].Year > years[j].Year
	})
	return &InvoiceArchiveResult{Years: years}, nil
}

func (s *Service) InvoiceArchiveFilePath(year, month int, filename string) (string, error) {
	if year <= 0 {
		return "", errors.New("invalid archive year")
	}
	if month < 1 || month > 12 {
		return "", errors.New("invalid archive month")
	}
	filename = strings.TrimSpace(filename)
	if !isArchivePDFName(filename) {
		return "", errors.New("invalid archive filename")
	}
	if filename != filepath.Base(filename) || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", errors.New("invalid archive filename")
	}

	fullPath := filepath.Join(
		s.rt.Dirs.Invoices,
		fmt.Sprintf("%04d", year),
		fmt.Sprintf("%02d", month),
		filename,
	)
	relPath, err := filepath.Rel(s.rt.Dirs.Invoices, fullPath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid archive filename")
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("invalid archive filename")
	}
	if info.IsDir() {
		return "", os.ErrNotExist
	}
	return fullPath, nil
}

type InvoiceArchiveZIPEntry struct {
	Name string
	Path string
}

func (s *Service) InvoiceArchiveZIPEntries(ctx context.Context, year, month int) ([]InvoiceArchiveZIPEntry, string, error) {
	if year <= 0 {
		return nil, "", errors.New("invalid archive year")
	}
	if month < 1 || month > 12 {
		return nil, "", errors.New("invalid archive month")
	}

	items, err := s.rt.DB.Ent.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(year),
			invoice.PeriodMonthEQ(month),
		).
		WithStudent().
		Order(ent.Asc(invoice.FieldID)).
		All(ctx)
	if err != nil {
		return nil, "", err
	}

	entries := make([]InvoiceArchiveZIPEntry, 0, len(items))
	for _, iv := range items {
		_, _, subjectName := archiveInvoiceNames(iv)
		info := s.invoicePDFInfo(iv, subjectName)
		if info.Status != invoiceArchivePDFStatusReady || info.Filename == "" || info.Path == "" {
			continue
		}
		entries = append(entries, InvoiceArchiveZIPEntry{
			Name: info.Filename,
			Path: info.Path,
		})
	}
	if len(entries) == 0 {
		return nil, "", errors.New("invalid archive zip: no ready pdfs for this month")
	}

	filename := fmt.Sprintf("rekini-%04d-%02d.zip", year, month)
	return entries, filename, nil
}

func (s *Service) invoiceArchiveInvoice(iv *ent.Invoice) InvoiceArchiveInvoiceDTO {
	studentName, recipientName, subjectName := archiveInvoiceNames(iv)
	item := InvoiceArchiveInvoiceDTO{
		InvoiceID:     iv.ID,
		Year:          iv.PeriodYear,
		Month:         iv.PeriodMonth,
		Number:        strings.TrimSpace(derefString(iv.Number)),
		StudentName:   studentName,
		RecipientName: recipientName,
		Total:         money.CentsToEuros(iv.TotalAmountCents),
		Status:        string(iv.Status),
		PDFStatus:     invoiceArchivePDFStatusMissing,
	}

	info := s.invoicePDFInfo(iv, subjectName)
	item.PDFStatus = info.Status
	if info.Filename != "" {
		item.PDFFilename = info.Filename
	}
	if info.GeneratedAt != nil && (info.Status == invoiceArchivePDFStatusReady || info.Status == invoiceArchivePDFStatusOutdated) {
		item.PDFUpdatedAt = info.GeneratedAt.UTC().Format(time.RFC3339)
	}
	if info.Status == invoiceArchivePDFStatusReady {
		item.OpenURL = invoiceArchiveFileURL(iv.PeriodYear, iv.PeriodMonth, info.Filename, "open")
		item.DownloadURL = invoiceArchiveFileURL(iv.PeriodYear, iv.PeriodMonth, info.Filename, "download")
	}

	return item
}

func archiveInvoiceNames(iv *ent.Invoice) (studentName, recipientName, subjectName string) {
	if iv == nil {
		return "", "", ""
	}
	if iv.Edges.Student != nil {
		studentName = strings.TrimSpace(iv.Edges.Student.FullName)
		recipientName = studentName
		subjectName = studentName
		if iv.Edges.Student.IsMinor {
			if childName := strings.TrimSpace(iv.Edges.Student.FullName); childName != "" {
				subjectName = childName
			}
			if payerName := strings.TrimSpace(iv.Edges.Student.PayerName); payerName != "" {
				recipientName = payerName
			}
		}
	}
	if recipientName == "" {
		recipientName = studentName
	}
	if subjectName == "" {
		subjectName = recipientName
	}
	return studentName, recipientName, subjectName
}

func (s *Service) invoicePDFInfo(iv *ent.Invoice, subjectName string) invsvc.PDFInfo {
	return invsvc.NewPDFLocator(s.rt.Dirs.Invoices).Evaluate(iv, subjectName)
}
