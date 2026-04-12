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

func GenerateInvoicePDFProfessional(ctx context.Context, db *ent.Client, invoiceID int, opt Options) (string, error) {
	iv, err := db.Invoice.Query().
		Where(invoice.IDEQ(invoiceID)).
		WithStudent().
		Only(ctx)
	if err != nil {
		return "", err
	}
	if iv.Number == nil || strings.TrimSpace(*iv.Number) == "" {
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
	_, err = normalizePath(opt.FontsDir) // keep validation so config errors are visible
	if err != nil {
		// We don't use fontsDir in this version, but keep the same option contract.
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

	labels := struct {
		Title         string
		InvoiceNo     string
		InvoiceDate   string
		ServicePeriod string
		Currency      string
		BillTo        string
		Payer         string

		Description string
		Quantity    string
		UnitPrice   string
		Amount      string

		Subtotal   string
		Total      string
		PayRef     string
		FooterNote string
	}{
		Title:         "INVOICE",
		InvoiceNo:     "Invoice No.",
		InvoiceDate:   "Invoice Date",
		ServicePeriod: "Service Period",
		Currency:      "Currency",
		BillTo:        "Bill To",
		Payer:         "Payer",

		Description: "Description",
		Quantity:    "Qty",
		UnitPrice:   "Unit Price",
		Amount:      "Amount",

		Subtotal:   "Subtotal",
		Total:      "Total",
		PayRef:     "Payment Reference",
		FooterNote: "This invoice was generated electronically.",
	}

	money := func(v float64) string {
		return fmt.Sprintf("%.2f %s", v, org.Currency)
	}

	// --- PDF ---
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(fmt.Sprintf("Invoice %s", *iv.Number), false)
	pdf.SetAuthor(org.Name, false)

	left, top, right := 16.0, 16.0, 16.0
	pdf.SetMargins(left, top, right)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AliasNbPages("")
	pdf.AddPage()

	// Footer
	pdf.SetFooterFunc(func() {
		pdf.SetY(-16)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(90, 90, 90)

		pdf.CellFormat(0, 4, fmt.Sprintf("%s: %s", labels.PayRef, *iv.Number), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 4, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
		pdf.Ln(5)
		pdf.CellFormat(0, 4, labels.FooterNote, "", 0, "L", false, 0, "")

		pdf.SetTextColor(0, 0, 0)
	})

	setGray := func() { pdf.SetTextColor(90, 90, 90) }
	setBlack := func() { pdf.SetTextColor(0, 0, 0) }

	// Title
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 8, labels.Title, "", 1, "L", false, 0, "")
	pdf.Ln(1)

	startY := pdf.GetY()
	pageW, _ := pdf.GetPageSize()
	contentW := pageW - left - right

	orgW := contentW * 0.58
	metaW := contentW - orgW
	gap := 4.0

	// Org block (left)
	pdf.SetXY(left, startY)
	pdf.SetFont("Helvetica", "B", 11)
	if strings.TrimSpace(org.Name) != "" {
		pdf.CellFormat(orgW-gap, 5.5, org.Name, "", 1, "L", false, 0, "")
	}
	pdf.SetFont("Helvetica", "", 10)
	if strings.TrimSpace(org.Address) != "" {
		pdf.SetXY(left, pdf.GetY())
		pdf.MultiCell(orgW-gap, 4.8, org.Address, "", "L", false)
	}

	// Meta box (right)
	metaX := left + orgW
	metaY := startY
	pdf.SetDrawColor(210, 210, 210)
	pdf.SetFillColor(248, 248, 248)
	boxH := 26.0
	pdf.Rect(metaX, metaY, metaW, boxH, "DF")

	pdf.SetXY(metaX+3, metaY+3)
	rowH := 5.6

	// TODO: use persisted issue date from DB if available
	invoiceDate := time.Now()
	dateStr := invoiceDate.Format("02.01.2006")
	periodStr := fmt.Sprintf("%04d-%02d", iv.PeriodYear, iv.PeriodMonth)

	kv := func(k, v string) {
		pdf.SetFont("Helvetica", "", 9)
		setGray()
		pdf.CellFormat(metaW*0.45, rowH, k, "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 9)
		setBlack()
		pdf.CellFormat(metaW*0.55-6, rowH, v, "", 1, "R", false, 0, "")
	}

	kv(labels.InvoiceNo, *iv.Number)
	kv(labels.InvoiceDate, dateStr)
	kv(labels.ServicePeriod, periodStr)
	kv(labels.Currency, org.Currency)

	afterHeaderY := startY + boxH + 6
	pdf.SetY(afterHeaderY)

	// Separator
	pdf.SetDrawColor(220, 220, 220)
	pdf.Line(left, pdf.GetY(), left+contentW, pdf.GetY())
	pdf.Ln(6)

	// Bill To
	clientName := ""
	if iv.Edges.Student != nil {
		clientName = strings.TrimSpace(iv.Edges.Student.FullName)
	}
	if clientName == "" {
		clientName = "—"
	}

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 6, labels.BillTo, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5.5, fmt.Sprintf("%s: %s", labels.Payer, clientName), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Table columns
	wDesc := contentW * 0.52
	wQty := contentW * 0.12
	wUnit := contentW * 0.18
	wAmt := contentW - wDesc - wQty - wUnit

	// Header row
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetDrawColor(210, 210, 210)

	cellHeader := func(w float64, txt, align string) {
		pdf.CellFormat(w, 8, txt, "1", 0, align, true, 0, "")
	}
	cellHeader(wDesc, labels.Description, "L")
	cellHeader(wQty, labels.Quantity, "R")
	cellHeader(wUnit, labels.UnitPrice, "R")
	cellHeader(wAmt, labels.Amount, "R")
	pdf.Ln(-1)

	// Rows (zebra)
	pdf.SetFont("Helvetica", "", 10)
	rowFill := false
	for _, l := range lines {
		if rowFill {
			pdf.SetFillColor(252, 252, 252)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		rowFill = !rowFill

		x0 := pdf.GetX()
		y0 := pdf.GetY()

		descLines := pdf.SplitLines([]byte(l.Description), wDesc-2)
		descH := float64(len(descLines)) * 5.2
		if descH < 7.0 {
			descH = 7.0
		}

		// Desc cell
		pdf.Rect(x0, y0, wDesc, descH, "DF")
		pdf.SetXY(x0+1, y0+1.2)
		pdf.MultiCell(wDesc-2, 5.2, l.Description, "", "L", false)

		// Qty
		pdf.SetXY(x0+wDesc, y0)
		pdf.Rect(pdf.GetX(), pdf.GetY(), wQty, descH, "DF")
		pdf.SetXY(x0+wDesc, y0)
		pdf.CellFormat(wQty, descH, fmt.Sprintf("%d", l.Qty), "", 0, "R", false, 0, "")

		// Unit
		pdf.SetXY(x0+wDesc+wQty, y0)
		pdf.Rect(pdf.GetX(), pdf.GetY(), wUnit, descH, "DF")
		pdf.SetXY(x0+wDesc+wQty, y0)
		pdf.CellFormat(wUnit, descH, money(l.UnitPrice), "", 0, "R", false, 0, "")

		// Amount
		pdf.SetXY(x0+wDesc+wQty+wUnit, y0)
		pdf.Rect(pdf.GetX(), pdf.GetY(), wAmt, descH, "DF")
		pdf.SetXY(x0+wDesc+wQty+wUnit, y0)
		pdf.CellFormat(wAmt, descH, money(l.Amount), "", 0, "R", false, 0, "")

		pdf.SetXY(x0, y0+descH)
	}

	pdf.Ln(6)

	// Totals box
	subtotal := iv.TotalAmount

	totW := contentW * 0.42
	totX := left + contentW - totW
	totY := pdf.GetY()

	pdf.SetDrawColor(210, 210, 210)
	pdf.SetFillColor(248, 248, 248)
	pdf.Rect(totX, totY, totW, 16, "DF")

	pdf.SetXY(totX+3, totY+3)
	lineH := 5.5
	labelW := totW * 0.45
	valueW := totW - labelW - 6

	pdf.SetFont("Helvetica", "", 10)
	setGray()
	pdf.CellFormat(labelW, lineH, labels.Subtotal, "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	setBlack()
	pdf.CellFormat(valueW, lineH, money(subtotal), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	setGray()
	pdf.CellFormat(labelW, lineH, labels.Total, "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	setBlack()
	pdf.CellFormat(valueW, lineH, money(iv.TotalAmount), "", 1, "R", false, 0, "")

	setBlack()

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return "", fmt.Errorf("write pdf %s: %w", outPath, err)
	}
	return outPath, nil
}
