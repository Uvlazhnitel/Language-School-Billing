package invoice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"langschool/ent"
)

const (
	PDFStatusReady    = "ready"
	PDFStatusMissing  = "missing"
	PDFStatusOutdated = "outdated"
	PDFStatusError    = "error"
)

type PDFLocator struct {
	baseDir string
}

type PDFDescriptor struct {
	CanonicalFilename string
	CanonicalPath     string
	LegacyPaths       []string
}

type PDFInfo struct {
	Status      string
	Filename    string
	Path        string
	GeneratedAt *time.Time
}

func NewPDFLocator(baseDir string) PDFLocator {
	return PDFLocator{baseDir: strings.TrimSpace(baseDir)}
}

func (l PDFLocator) PathByNumber(y, m int, number string) string {
	return filepath.Join(l.baseDir, yearMonthDir(y, m), sanitizeInvoiceFileName(number)+".pdf")
}

func (l PDFLocator) PathByNumberAndName(y, m int, number, subjectName string) string {
	return filepath.Join(l.baseDir, yearMonthDir(y, m), invoiceFileStem(number, subjectName)+".pdf")
}

func (l PDFLocator) CanonicalPath(y, m int, filename string) string {
	filename = strings.TrimSpace(filepath.Base(filename))
	if filename == "" || filename == "." {
		return ""
	}
	return filepath.Join(l.baseDir, yearMonthDir(y, m), filename)
}

func (l PDFLocator) Describe(iv *ent.Invoice, subjectName string) PDFDescriptor {
	if iv == nil {
		return PDFDescriptor{}
	}
	desc := PDFDescriptor{
		CanonicalFilename: strings.TrimSpace(stringValue(iv.PdfFilename)),
	}
	if desc.CanonicalFilename != "" {
		desc.CanonicalPath = l.CanonicalPath(iv.PeriodYear, iv.PeriodMonth, desc.CanonicalFilename)
	}
	if iv.Number != nil {
		number := strings.TrimSpace(*iv.Number)
		if number != "" {
			desc.LegacyPaths = []string{
				l.PathByNumberAndName(iv.PeriodYear, iv.PeriodMonth, number, subjectName),
				l.PathByNumber(iv.PeriodYear, iv.PeriodMonth, number),
			}
		}
	}
	return desc
}

func (l PDFLocator) Evaluate(iv *ent.Invoice, subjectName string) PDFInfo {
	if iv == nil || iv.Number == nil || strings.TrimSpace(*iv.Number) == "" {
		return PDFInfo{Status: PDFStatusMissing}
	}
	desc := l.Describe(iv, subjectName)
	if desc.CanonicalFilename != "" {
		if desc.CanonicalPath == "" {
			return PDFInfo{Status: PDFStatusError}
		}
		_, err := os.Stat(desc.CanonicalPath)
		if err != nil {
			if os.IsNotExist(err) {
				return PDFInfo{Status: PDFStatusMissing}
			}
			return PDFInfo{Status: PDFStatusError}
		}
		pdfInfo := PDFInfo{
			Filename: desc.CanonicalFilename,
			Path:     desc.CanonicalPath,
		}
		if iv.PdfGeneratedAt != nil && !iv.PdfGeneratedAt.IsZero() {
			generatedAt := iv.PdfGeneratedAt.UTC()
			pdfInfo.GeneratedAt = &generatedAt
		}
		if iv.PdfRevision != nil && *iv.PdfRevision == iv.Version {
			pdfInfo.Status = PDFStatusReady
			return pdfInfo
		}
		pdfInfo.Status = PDFStatusOutdated
		return pdfInfo
	}
	if iv.PdfRevision != nil || iv.PdfGeneratedAt != nil {
		return PDFInfo{Status: PDFStatusError}
	}
	for _, path := range desc.LegacyPaths {
		_, err := os.Stat(path)
		if err == nil {
			return PDFInfo{
				Status:   PDFStatusOutdated,
				Filename: filepath.Base(path),
				Path:     path,
			}
		}
		if !os.IsNotExist(err) {
			return PDFInfo{Status: PDFStatusError}
		}
	}
	return PDFInfo{Status: PDFStatusMissing}
}

func yearMonthDir(y, m int) string {
	return filepath.Join(
		fmt.Sprintf("%04d", y),
		fmt.Sprintf("%02d", m),
	)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
