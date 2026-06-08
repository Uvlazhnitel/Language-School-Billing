package ent

import "langschool/internal/money"

func (c *CourseCreate) SetLessonPrice(v float64) *CourseCreate {
	return c.SetLessonPriceCents(money.EurosToCents(v))
}

func (c *CourseCreate) SetSubscriptionPrice(v float64) *CourseCreate {
	return c.SetSubscriptionPriceCents(money.EurosToCents(v))
}

func (u *CourseUpdateOne) SetLessonPrice(v float64) *CourseUpdateOne {
	return u.SetLessonPriceCents(money.EurosToCents(v))
}

func (u *CourseUpdateOne) SetSubscriptionPrice(v float64) *CourseUpdateOne {
	return u.SetSubscriptionPriceCents(money.EurosToCents(v))
}

func (c *InvoiceCreate) SetTotalAmount(v float64) *InvoiceCreate {
	return c.SetTotalAmountCents(money.EurosToCents(v))
}

func (u *InvoiceUpdateOne) SetTotalAmount(v float64) *InvoiceUpdateOne {
	return u.SetTotalAmountCents(money.EurosToCents(v))
}

func (c *InvoiceLineCreate) SetUnitPrice(v float64) *InvoiceLineCreate {
	return c.SetUnitPriceCents(money.EurosToCents(v))
}

func (c *InvoiceLineCreate) SetAmount(v float64) *InvoiceLineCreate {
	return c.SetAmountCents(money.EurosToCents(v))
}

func (c *PaymentCreate) SetAmount(v float64) *PaymentCreate {
	return c.SetAmountCents(money.EurosToCents(v))
}

func (u *PaymentUpdateOne) SetAmount(v float64) *PaymentUpdateOne {
	return u.SetAmountCents(money.EurosToCents(v))
}
