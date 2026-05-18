package recipient

import (
	"context"
	"strings"

	"langschool/ent"
	"langschool/ent/student"
)

// Info describes who should appear as the visible recipient of a student's invoice/reminder.
type Info struct {
	RecipientName       string
	RecipientPhone      string
	RecipientEmail      string
	ChildName           string
	StudentPersonalCode string
	IsMinor             bool
}

// InvoiceSubjectName returns the student-facing name that should appear in an
// invoice title or file name. For minors, prefer the child's name; otherwise
// use the visible recipient/adult name.
func (i Info) InvoiceSubjectName() string {
	if i.IsMinor && strings.TrimSpace(i.ChildName) != "" {
		return strings.TrimSpace(i.ChildName)
	}
	if strings.TrimSpace(i.RecipientName) != "" {
		return strings.TrimSpace(i.RecipientName)
	}
	return strings.TrimSpace(i.ChildName)
}

// ResolveInvoiceRecipient determines the visible invoice recipient for a student.
// Invoices still belong to students in the database; this helper only affects display output.
func ResolveInvoiceRecipient(ctx context.Context, db *ent.Client, studentID int) (Info, error) {
	st, err := db.Student.Query().
		Where(student.IDEQ(studentID)).
		Only(ctx)
	if err != nil {
		return Info{}, err
	}

	info := Info{
		RecipientName:       st.FullName,
		RecipientPhone:      st.Phone,
		RecipientEmail:      st.Email,
		ChildName:           st.FullName,
		StudentPersonalCode: st.PersonalCode,
		IsMinor:             st.IsMinor,
	}
	if !st.IsMinor {
		return info, nil
	}
	if st.PayerName != "" {
		info.RecipientName = st.PayerName
	}
	return info, nil
}
