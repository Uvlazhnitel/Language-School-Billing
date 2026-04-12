package pdf

import (
	"context"

	"langschool/ent"
)

// GenerateInvoicePDF is kept for backward compatibility.
// It now uses the professional layout.
func GenerateInvoicePDF(ctx context.Context, db *ent.Client, invoiceID int, opt Options) (string, error) {
	return GenerateInvoicePDFProfessional(ctx, db, invoiceID, opt)
}