// internal/pdf/invoice_pdf.go

package pdf

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-pdf/fpdf"

	"langschool/ent"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/settings"
	"langschool/internal/app"
	"langschool/internal/app/recipient"
)

type artlabProvider struct {
	DisplayName       string
	LegalName         string
	RegistrationNo    string
	LegalAddress      string
	StructuralUnit    string
	StructuralUnitReg string
	Phone             string
	ContactPerson     string
	Bank              string
	Swift             string
	IBAN              string
	Currency          string
	Locale            string
}

func artlabProviderDefaults() artlabProvider {
	return artlabProvider{
		DisplayName:       "ArtLab",
		LegalName:         "Biedrība „Kultūras, mākslas un izglītības centrs ARTLAB”",
		RegistrationNo:    "40008216321",
		LegalAddress:      "Latgales iela 260, Rīga, LV-1063",
		StructuralUnit:    "Interešu izglītības iestāde „Avots”",
		StructuralUnitReg: "3351803284",
		Phone:             "26130586",
		ContactPerson:     "Svetlana Labuta",
		Bank:              "A/S SEB banka",
		Swift:             "UNLALV2X",
		IBAN:              "LV92UNLA0050021521167",
		Currency:          "EUR",
		Locale:            "lv-LV",
	}
}

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

	provider := artlabProviderDefaults()

	st, _ := db.Settings.Query().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).Only(ctx)
	if st != nil {
		if strings.TrimSpace(st.OrgName) != "" {
			provider.DisplayName = st.OrgName
		}
		if strings.TrimSpace(st.Address) != "" {
			provider.LegalAddress = st.Address
		}
		if strings.TrimSpace(st.Currency) != "" {
			provider.Currency = st.Currency
		}
		if strings.TrimSpace(st.Locale) != "" {
			provider.Locale = st.Locale
		}
	}
	if strings.TrimSpace(opt.Currency) != "" {
		provider.Currency = opt.Currency
	}
	if strings.TrimSpace(opt.Locale) != "" {
		provider.Locale = opt.Locale
	}

	outBase, err := normalizePath(opt.OutBaseDir)
	if err != nil {
		return "", fmt.Errorf("OutBaseDir: %w", err)
	}
	fontsDir, err := normalizePath(opt.FontsDir)
	if err != nil {
		return "", fmt.Errorf("FontsDir: %w", err)
	}

	lines, err := db.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		All(ctx)
	if err != nil {
		return "", err
	}

	year, month := iv.PeriodYear, iv.PeriodMonth
	dir := filepath.Join(outBase, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month))
	if err := os.MkdirAll(dir, app.DirPermission); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}

	recipientInfo, err := recipient.ResolveInvoiceRecipient(ctx, db, iv.StudentID)
	if err != nil {
		return "", err
	}
	subjectName := recipientInfo.InvoiceSubjectName()
	outPath := filepath.Join(dir, fmt.Sprintf("%s.pdf", invoiceFileStem(*iv.Number, subjectName)))

	invoiceDate := time.Now()
	dueDate := invoiceDate.AddDate(0, 0, 14)

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	periodEnd := periodStart.AddDate(0, 1, -1)

	p := fpdf.New("P", "mm", "A4", "")
	p.SetTitle(fmt.Sprintf("Rēķins %s — %s", *iv.Number, subjectName), false)
	p.SetAuthor(provider.DisplayName, false)
	p.SetMargins(10, 10, 10)
	p.SetAutoPageBreak(true, 18)
	p.AliasNbPages("")

	if err := addArtLabFonts(p, fontsDir); err != nil {
		return "", err
	}

	p.SetFooterFunc(func() {
		p.SetY(-13)
		p.SetDrawColor(220, 220, 220)
		p.Line(10, p.GetY(), 200, p.GetY())
		p.Ln(2)

		p.SetFont("DejaVu", "", 7.5)
		p.SetTextColor(100, 100, 100)
		p.CellFormat(95, 4, fmt.Sprintf("Maksājuma mērķis: %s", *iv.Number), "", 0, "L", false, 0, "")
		p.CellFormat(95, 4, fmt.Sprintf("Lapa %d/{nb}", p.PageNo()), "", 0, "R", false, 0, "")
		p.SetTextColor(0, 0, 0)
	})

	p.AddPage()

	drawHeader(p, provider, *iv.Number, invoiceDate)
	drawProviderBlock(p, provider)
	drawRecipientBlock(p, recipientInfo)
	drawServiceTable(p, provider.Currency, lines, periodStart, periodEnd)
	drawTotalAndPayment(p, provider, iv.TotalAmount, dueDate, *iv.Number)

	if err := p.OutputFileAndClose(outPath); err != nil {
		return "", fmt.Errorf("write pdf %s: %w", outPath, err)
	}

	return outPath, nil
}

func addArtLabFonts(p *fpdf.Fpdf, fontsDir string) error {
	regular := filepath.Join(fontsDir, "DejaVuSans.ttf")
	bold := filepath.Join(fontsDir, "DejaVuSans-Bold.ttf")
	italic := filepath.Join(fontsDir, "DejaVuSans-Oblique.ttf")
	boldItalic := filepath.Join(fontsDir, "DejaVuSans-BoldOblique.ttf")

	required := []string{regular, bold}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("font file not found: %s", path)
		}
	}

	p.AddUTF8Font("DejaVu", "", regular)
	p.AddUTF8Font("DejaVu", "B", bold)

	if _, err := os.Stat(italic); err == nil {
		p.AddUTF8Font("DejaVu", "I", italic)
	} else {
		p.AddUTF8Font("DejaVu", "I", regular)
	}

	if _, err := os.Stat(boldItalic); err == nil {
		p.AddUTF8Font("DejaVu", "BI", boldItalic)
	} else {
		p.AddUTF8Font("DejaVu", "BI", bold)
	}

	return nil
}

func drawHeader(p *fpdf.Fpdf, provider artlabProvider, number string, invoiceDate time.Time) {
	left := 10.0
	right := 200.0

	p.SetTextColor(20, 20, 20)

	p.SetFont("DejaVu", "B", 26)
	p.CellFormat(110, 10, provider.DisplayName, "", 0, "L", false, 0, "")

	p.SetFont("DejaVu", "B", 15)
	p.CellFormat(80, 10, "RĒĶINS", "", 1, "R", false, 0, "")

	p.SetFont("DejaVu", "I", 8.5)
	p.SetX(left + 17)
	p.CellFormat(95, 5, "Kultūras, mākslas un izglītības centrs", "", 0, "L", false, 0, "")

	p.SetFont("DejaVu", "", 9)
	p.SetX(130)
	p.CellFormat(28, 5, "Rēķins Nr.", "", 0, "L", false, 0, "")
	p.SetFont("DejaVu", "B", 9)
	p.CellFormat(42, 5, number, "B", 1, "C", false, 0, "")

	p.SetFont("DejaVu", "", 9)
	p.SetX(130)
	p.CellFormat(28, 5, "Datums", "", 0, "L", false, 0, "")
	p.SetFont("DejaVu", "B", 9)
	p.CellFormat(42, 5, invoiceDate.Format("02.01.2006"), "B", 1, "C", false, 0, "")

	p.Ln(10)

	p.SetDrawColor(40, 40, 40)
	p.SetLineWidth(0.35)
	p.Line(left, p.GetY(), right, p.GetY())
	p.SetLineWidth(0.2)
}

func drawProviderBlock(p *fpdf.Fpdf, provider artlabProvider) {
	p.Ln(2)
	sectionTitle(p, "PAKALPOJUMA SNIEDZĒJS")

	rows := []struct {
		label string
		value string
	}{
		{"Nosaukums", provider.LegalName},
		{"Reģ. Nr.", provider.RegistrationNo},
		{"Juridiskā adrese", provider.LegalAddress},
		{"Struktūrvienība", fmt.Sprintf("%s    Reģ. Nr. %s", provider.StructuralUnit, provider.StructuralUnitReg)},
		{"Tālrunis", fmt.Sprintf("%s %s", provider.ContactPerson, provider.Phone)},
		{"Banka", fmt.Sprintf("%s, %s", provider.Bank, provider.Swift)},
		{"Konts", provider.IBAN},
	}

	infoTable(p, rows)
	p.Ln(5)
}

func drawRecipientBlock(p *fpdf.Fpdf, recipient recipient.Info) {
	sectionTitle(p, "PAKALPOJUMA SAŅĒMĒJS")

	rows := []struct {
		label string
		value string
	}{
		{"Vārds, uzvārds / Nosaukums", recipient.RecipientName},
		{"Personas kods / Reģ. Nr.", func() string {
			if recipient.IsMinor {
				return ""
			}
			return recipient.StudentPersonalCode
		}()},
		{"E-pasts / tālrunis", strings.TrimSpace(strings.Join(filterNonEmptyStrings(recipient.RecipientEmail, recipient.RecipientPhone), " / "))},
	}
	if recipient.IsMinor {
		rows = append(rows, struct {
			label string
			value string
		}{
			label: "Bērns / audzēknis",
			value: recipient.ChildName,
		}, struct {
			label string
			value string
		}{
			label: "Bērna personas kods",
			value: recipient.StudentPersonalCode,
		})
	}

	infoTable(p, rows)
	p.Ln(5)
}

func filterNonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func drawServiceTable(p *fpdf.Fpdf, currency string, lines []*ent.InvoiceLine, periodStart, periodEnd time.Time) {
	sectionTitle(p, "PAKALPOJUMA RAKSTUROJUMS")

	x := 10.0
	wNo := 11.0
	wName := 67.0
	wTerm := 37.0
	wUnit := 20.0
	wQty := 18.0
	wPrice := 19.0
	wAmount := 18.0
	headerH := 7.0

	p.SetX(x)
	p.SetFont("DejaVu", "B", 7.5)
	p.SetFillColor(242, 244, 247)
	p.SetDrawColor(70, 70, 70)

	tableHeaderCell(p, wNo, headerH, "Nr.p.k.", "C")
	tableHeaderCell(p, wName, headerH, "Nosaukums", "C")
	tableHeaderCell(p, wTerm, headerH, "Termiņš", "C")
	tableHeaderCell(p, wUnit, headerH, "Mērv.", "C")
	tableHeaderCell(p, wQty, headerH, "Daudzums", "C")
	tableHeaderCell(p, wPrice, headerH, "Cena", "C")
	tableHeaderCell(p, wAmount, headerH, "Vērtība", "C")
	p.Ln(-1)

	p.SetFont("DejaVu", "", 8)
	period := fmt.Sprintf("%s–%s", periodStart.Format("02.01.2006"), periodEnd.Format("02.01.2006"))

	if len(lines) == 0 {
		drawServiceRow(p, 1, "Mācību pakalpojumi", period, "gab.", 1, 0, 0, currency)
		return
	}

	for i, line := range lines {
		unit := "gab."
		if line.Qty > 1 {
			unit = "nod."
		}

		description := normalizeInvoiceDescription(line.Description, periodStart)
		drawServiceRow(p, i+1, description, period, unit, line.Qty, line.UnitPrice, line.Amount, currency)
	}
}

func drawServiceRow(p *fpdf.Fpdf, no int, name, term, unit string, qty int, price, amount float64, currency string) {
	x0 := 10.0
	y0 := p.GetY()

	wNo := 11.0
	wName := 67.0
	wTerm := 37.0
	wUnit := 20.0
	wQty := 18.0
	wPrice := 19.0
	wAmount := 18.0

	descLines := p.SplitLines([]byte(name), wName-3)
	rowH := math.Max(10, float64(len(descLines))*4.2+4)

	if y0+rowH > 268 {
		p.AddPage()
		y0 = p.GetY()
	}

	p.SetDrawColor(70, 70, 70)
	p.SetFillColor(255, 255, 255)

	cellRect(p, x0, y0, wNo, rowH)
	cellRect(p, x0+wNo, y0, wName, rowH)
	cellRect(p, x0+wNo+wName, y0, wTerm, rowH)
	cellRect(p, x0+wNo+wName+wTerm, y0, wUnit, rowH)
	cellRect(p, x0+wNo+wName+wTerm+wUnit, y0, wQty, rowH)
	cellRect(p, x0+wNo+wName+wTerm+wUnit+wQty, y0, wPrice, rowH)
	cellRect(p, x0+wNo+wName+wTerm+wUnit+wQty+wPrice, y0, wAmount, rowH)

	p.SetFont("DejaVu", "B", 8)
	p.SetXY(x0, y0+rowH-5.2)
	p.CellFormat(wNo, 4, fmt.Sprintf("%d.", no), "", 0, "C", false, 0, "")

	p.SetFont("DejaVu", "", 8)
	p.SetXY(x0+wNo+1.5, y0+2)
	p.MultiCell(wName-3, 4.2, name, "", "L", false)

	p.SetXY(x0+wNo+wName, y0+(rowH/2)-2)
	p.CellFormat(wTerm, 4, term, "", 0, "C", false, 0, "")

	p.SetXY(x0+wNo+wName+wTerm, y0+(rowH/2)-2)
	p.CellFormat(wUnit, 4, unit, "", 0, "C", false, 0, "")

	p.SetXY(x0+wNo+wName+wTerm+wUnit, y0+(rowH/2)-2)
	p.CellFormat(wQty, 4, fmt.Sprintf("%d", qty), "", 0, "C", false, 0, "")

	p.SetXY(x0+wNo+wName+wTerm+wUnit+wQty, y0+(rowH/2)-2)
	p.CellFormat(wPrice-1.5, 4, moneyNoCurrency(price), "", 0, "R", false, 0, "")

	p.SetXY(x0+wNo+wName+wTerm+wUnit+wQty+wPrice, y0+(rowH/2)-2)
	p.CellFormat(wAmount-1.5, 4, moneyNoCurrency(amount), "", 0, "R", false, 0, "")

	p.SetY(y0 + rowH)
	_ = currency
}

func drawTotalAndPayment(p *fpdf.Fpdf, provider artlabProvider, total float64, dueDate time.Time, invoiceNumber string) {
	p.Ln(0)

	totalBoxW := 54.0
	totalBoxX := 146.0
	y := p.GetY()

	p.SetDrawColor(70, 70, 70)
	p.Rect(totalBoxX, y, totalBoxW, 11, "D")
	p.Line(totalBoxX+31, y, totalBoxX+31, y+11)

	p.SetFont("DejaVu", "B", 8)
	p.SetXY(totalBoxX+2, y+3.2)
	p.CellFormat(28, 4, "Kopā EUR:", "", 0, "L", false, 0, "")

	p.SetFont("DejaVu", "B", 9)
	p.SetXY(totalBoxX+32, y+3.2)
	p.CellFormat(20, 4, moneyNoCurrency(total), "", 0, "R", false, 0, "")

	p.SetY(y + 19)

	p.SetFont("DejaVu", "I", 8)
	p.CellFormat(54, 5, "Samaksas veids un kārtība:", "", 0, "R", false, 0, "")
	p.SetTextColor(170, 0, 0)
	p.SetFont("DejaVu", "B", 9)
	p.CellFormat(54, 5, dueDate.Format("02.01.2006"), "B", 1, "C", false, 0, "")
	p.SetTextColor(0, 0, 0)

	p.Ln(3)

	euro, cents := splitEUR(total)
	words := fmt.Sprintf("%s eiro un %02d centi", capitalizeFirst(latvianNumberWords(euro)), cents)

	p.SetFont("DejaVu", "I", 8)
	p.CellFormat(54, 5, "Summa vārdiem:", "", 0, "R", false, 0, "")
	p.SetFont("DejaVu", "B", 8.5)
	p.CellFormat(136, 5, words, "B", 1, "C", false, 0, "")

	p.Ln(7)

	p.SetFillColor(248, 249, 251)
	p.SetDrawColor(210, 210, 210)
	p.Rect(10, p.GetY(), 190, 28, "DF")

	p.SetXY(13, p.GetY()+3)
	p.SetFont("DejaVu", "B", 9)
	p.CellFormat(184, 5, "Maksājuma informācija", "", 1, "L", false, 0, "")

	p.SetFont("DejaVu", "", 8)
	p.SetX(13)
	p.CellFormat(36, 4.5, "Saņēmējs:", "", 0, "L", false, 0, "")
	p.CellFormat(145, 4.5, provider.LegalName, "", 1, "L", false, 0, "")

	p.SetX(13)
	p.CellFormat(36, 4.5, "Banka:", "", 0, "L", false, 0, "")
	p.CellFormat(145, 4.5, fmt.Sprintf("%s, SWIFT: %s", provider.Bank, provider.Swift), "", 1, "L", false, 0, "")

	p.SetX(13)
	p.CellFormat(36, 4.5, "Konts:", "", 0, "L", false, 0, "")
	p.SetFont("DejaVu", "B", 8)
	p.CellFormat(145, 4.5, provider.IBAN, "", 1, "L", false, 0, "")

	p.SetFont("DejaVu", "", 8)
	p.SetX(13)
	p.CellFormat(36, 4.5, "Maksājuma mērķis:", "", 0, "L", false, 0, "")
	p.SetFont("DejaVu", "B", 8)
	p.CellFormat(145, 4.5, invoiceNumber, "", 1, "L", false, 0, "")

	p.Ln(9)
	p.SetFont("DejaVu", "B", 8.5)
	p.CellFormat(190, 5, "Ja maksājums netiek veikts līdz norādītajam termiņam, rēķins var tikt anulēts.", "", 1, "C", false, 0, "")

	p.Ln(6)
	p.SetFont("DejaVu", "", 7.5)
	p.SetTextColor(90, 90, 90)
	p.MultiCell(190, 4, "Rēķins ir sagatavots elektroniski un ir derīgs bez paraksta.", "", "C", false)
	p.SetTextColor(0, 0, 0)
}

func sectionTitle(p *fpdf.Fpdf, title string) {
	p.SetFont("DejaVu", "B", 10)
	p.SetFillColor(255, 255, 255)
	p.CellFormat(190, 6, title, "", 1, "L", false, 0, "")
	p.SetDrawColor(50, 50, 50)
	p.Line(10, p.GetY(), 200, p.GetY())
}

func infoTable(p *fpdf.Fpdf, rows []struct {
	label string
	value string
}) {
	labelW := 67.0
	valueW := 123.0
	rowH := 5.5

	p.SetDrawColor(70, 70, 70)

	for _, row := range rows {
		y := p.GetY()
		p.Rect(10, y, labelW, rowH, "D")
		p.Rect(10+labelW, y, valueW, rowH, "D")

		p.SetFont("DejaVu", "I", 7.5)
		p.SetXY(11, y+1.1)
		p.CellFormat(labelW-2, 3.8, row.label, "", 0, "L", false, 0, "")

		p.SetFont("DejaVu", "B", 8)
		p.SetXY(10+labelW+2, y+1.1)
		p.CellFormat(valueW-4, 3.8, row.value, "", 0, "C", false, 0, "")

		p.SetY(y + rowH)
	}
}

func tableHeaderCell(p *fpdf.Fpdf, w, h float64, text, align string) {
	p.CellFormat(w, h, text, "1", 0, align, true, 0, "")
}

func cellRect(p *fpdf.Fpdf, x, y, w, h float64) {
	p.Rect(x, y, w, h, "D")
}

func normalizeInvoiceDescription(description string, periodStart time.Time) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return fmt.Sprintf("Mācību pakalpojumi — %s", latvianMonthName(periodStart.Month()))
	}

	return description
}

func latvianMonthName(month time.Month) string {
	names := []string{
		"janvāris",
		"februāris",
		"marts",
		"aprīlis",
		"maijs",
		"jūnijs",
		"jūlijs",
		"augusts",
		"septembris",
		"oktobris",
		"novembris",
		"decembris",
	}

	i := int(month) - 1
	if i < 0 || i >= len(names) {
		return ""
	}

	return names[i]
}

func moneyNoCurrency(v float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", v), ".", ",")
}

func splitEUR(v float64) (int, int) {
	totalCents := int(math.Round(v * 100))
	return totalCents / 100, totalCents % 100
}

func safeFileName(name string) string {
	name = strings.TrimSpace(name)

	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)

	return replacer.Replace(name)
}

func invoiceFileStem(number, subjectName string) string {
	subjectName = strings.TrimSpace(subjectName)
	if subjectName == "" {
		return safeFileName(number)
	}
	return safeFileName(fmt.Sprintf("%s - %s", number, subjectName))
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}

	return strings.ToUpper(string(r)) + s[size:]
}

func latvianNumberWords(n int) string {
	if n == 0 {
		return "nulle"
	}

	if n < 0 {
		return "mīnus " + latvianNumberWords(-n)
	}

	if n < 20 {
		return []string{
			"nulle",
			"viens",
			"divi",
			"trīs",
			"četri",
			"pieci",
			"seši",
			"septiņi",
			"astoņi",
			"deviņi",
			"desmit",
			"vienpadsmit",
			"divpadsmit",
			"trīspadsmit",
			"četrpadsmit",
			"piecpadsmit",
			"sešpadsmit",
			"septiņpadsmit",
			"astoņpadsmit",
			"deviņpadsmit",
		}[n]
	}

	if n < 100 {
		tens := n / 10
		rem := n % 10

		tensWords := map[int]string{
			2: "divdesmit",
			3: "trīsdesmit",
			4: "četrdesmit",
			5: "piecdesmit",
			6: "sešdesmit",
			7: "septiņdesmit",
			8: "astoņdesmit",
			9: "deviņdesmit",
		}

		if rem == 0 {
			return tensWords[tens]
		}

		return tensWords[tens] + " " + latvianNumberWords(rem)
	}

	if n < 1000 {
		hundreds := n / 100
		rem := n % 100

		var prefix string
		if hundreds == 1 {
			prefix = "simts"
		} else {
			prefix = latvianNumberWords(hundreds) + " simti"
		}

		if rem == 0 {
			return prefix
		}

		return prefix + " " + latvianNumberWords(rem)
	}

	if n < 1000000 {
		thousands := n / 1000
		rem := n % 1000

		var prefix string
		if thousands == 1 {
			prefix = "viens tūkstotis"
		} else {
			prefix = latvianNumberWords(thousands) + " tūkstoši"
		}

		if rem == 0 {
			return prefix
		}

		return prefix + " " + latvianNumberWords(rem)
	}

	return fmt.Sprintf("%d", n)
}
