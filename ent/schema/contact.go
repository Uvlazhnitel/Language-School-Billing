package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Contact struct{ ent.Schema }

func (Contact) Fields() []ent.Field {
	return []ent.Field{
		field.String("full_name"),
		field.String("phone").Default(""),
		field.String("email").Default(""),
		field.String("note").Default(""),
		field.Bool("is_active").Default(true),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Contact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("student_contacts", StudentContact.Type),
	}
}
