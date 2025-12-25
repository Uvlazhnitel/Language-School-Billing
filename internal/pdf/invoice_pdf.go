// internal/pdf/invoice_pdf.go
package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"langschool/ent"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/settings"
	"langschool/internal/app"
)

// Options — where fonts are located and where to save the PDF.
type Options struct {
	OutBaseDir string // root folder Invoices/
	FontsDir   string // folder with TTF: DejaVuSans.ttf, DejaVuSans-Bold.ttf
	Currency   string // "EUR"
	Locale     string
}

// normalizePath ensures the path is absolute and clean
func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	// Handle macOS-specific case where paths might be missing leading slash
	if strings.HasPrefix(path, "Users/") {
		path = "/" + path
	}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return filepath.Clean(path), nil
}

// GenerateInvoicePDF creates a PDF for an already NUMBERED invoice (status=issued).
// Returns the full path to the PDF.
func GenerateInvoicePDF(ctx context.Context, db *ent.Client, invoiceID int, opt Options) (string, error) {
	iv, err := db.Invoice.Query().
		Where(invoice.IDEQ(invoiceID)).
		WithStudent().
		Only(ctx)
	if err != nil {
		return "", err
	}
	if iv.Number == nil || *iv.Number == "" {
		return "", fmt.Errorf("invoice %d has no number (issue it first)", invoiceID)
	}

	// --- Organization settings ---
	st, _ := db.Settings.Query().Where(settings.SingletonIDEQ(1)).Only(ctx)
	org := struct {
		Name, Address, Prefix, Currency, Locale string
	}{
		Name:     "",
		Address:  "",
		Prefix:   "",
		Currency: "EUR",
		Locale:   "lv-LV",
	}
	if st != nil {
		org.Name = st.OrgName
		org.Address = st.Address
		org.Prefix = st.InvoicePrefix
		org.Currency = st.Currency
		org.Locale = st.Locale
	}
	if opt.Currency != "" {
		org.Currency = opt.Currency
	}
	if opt.Locale != "" {
		org.Locale = opt.Locale
	}

	// --- Path normalization ---
	outBase, err := normalizePath(opt.OutBaseDir)
	if err != nil {
		return "", fmt.Errorf("OutBaseDir: %w", err)
	}

	fontsDir, err := normalizePath(opt.FontsDir)
	if err != nil {
		return "", fmt.Errorf("FontsDir: %w", err)
	}

	// --- Lines ---
	lines, err := db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).All(ctx)
	if err != nil {
		return "", err
	}

	// --- Output path: YYYY/MM/NUMBER.pdf ---
	year, month := iv.PeriodYear, iv.PeriodMonth
	dir := filepath.Join(outBase, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month))
	if err := os.MkdirAll(dir, app.DirPermission); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}
	outPath := filepath.Join(dir, fmt.Sprintf("%s.pdf", *iv.Number))

	// --- Fonts (absolute) ---
	reg := filepath.Join(fontsDir, "DejaVuSans.ttf")
	bld := filepath.Join(fontsDir, "DejaVuSans-Bold.ttf")
	if _, err := os.Stat(reg); err != nil {
		return "", fmt.Errorf("font not found: %s (%w)", reg, err)
	}
	if _, err := os.Stat(bld); err != nil {
		return "", fmt.Errorf("font not found: %s (%w)", bld, err)
	}

	// --- PDF ---
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font("DejaVu", "", reg)
	pdf.AddUTF8Font("DejaVu", "B", bld)

	pdf.SetMargins(16, 16, 16)
	pdf.AddPage()

	// Header
	pdf.SetFont("DejaVu", "B", 16)
	pdf.CellFormat(0, 8, "INVOICE", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 11)
	num := ""
	if iv.Number != nil {
		num = *iv.Number
	}
	dateStr := time.Now().Format("02.01.2006")
	pdf.CellFormat(0, 6, fmt.Sprintf("Number: %s    Date: %s", num, dateStr), "", 1, "L", false, 0, "")

	// Organization
	if org.Name != "" {
		pdf.SetFont("DejaVu", "B", 11)
		pdf.CellFormat(0, 6, org.Name, "", 1, "L", false, 0, "")
	}
	if org.Address != "" {
		pdf.SetFont("DejaVu", "", 10)
		pdf.MultiCell(0, 5, org.Address, "", "L", false)
	}

	// Client
	pdf.Ln(2)
	clientName := ""
	if iv.Edges.Student != nil {
		clientName = iv.Edges.Student.FullName
	}
	pdf.SetFont("DejaVu", "", 11)
	pdf.CellFormat(0, 6, fmt.Sprintf("Payer: %s", clientName), "", 1, "L", false, 0, "")

	// Table
	pdf.Ln(2)
	pdf.SetFont("DejaVu", "B", 10)
	w := []float64{90, 20, 30, 30}
	headers := []string{"Description", "Quantity", "Price", "Amount"}
	for i, h := range headers {
		align := "L"
		if i > 0 {
			align = "R"
		}
		pdf.CellFormat(w[i], 7, h, "TB", 0, align, false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("DejaVu", "", 10)
	for _, l := range lines {
		pdf.CellFormat(w[0], 6, l.Description, "B", 0, "L", false, 0, "")
		pdf.CellFormat(w[1], 6, fmt.Sprintf("%d", l.Qty), "B", 0, "R", false, 0, "")
		pdf.CellFormat(w[2], 6, fmt.Sprintf("%.2f %s", l.UnitPrice, org.Currency), "B", 0, "R", false, 0, "")
		pdf.CellFormat(w[3], 6, fmt.Sprintf("%.2f %s", l.Amount, org.Currency), "B", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	// Total
	pdf.Ln(2)
	pdf.SetFont("DejaVu", "B", 11)
	pdf.CellFormat(w[0]+w[1]+w[2], 7, "TOTAL:", "", 0, "R", false, 0, "")
	pdf.CellFormat(w[3], 7, fmt.Sprintf("%.2f %s", iv.TotalAmount, org.Currency), "", 1, "R", false, 0, "")

	// Footer
	pdf.SetY(-26)
	pdf.SetFont("DejaVu", "", 9)
	pdf.CellFormat(0, 5, "Thank you for your timely payment!", "", 1, "C", false, 0, "")

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return "", fmt.Errorf("write pdf %s: %w", outPath, err)
	}
	return outPath, nil
}
