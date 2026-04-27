package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type StudentContact struct{ ent.Schema }

func (StudentContact) Fields() []ent.Field {
	return []ent.Field{
		field.Int("student_id"),
		field.Int("contact_id"),
		field.String("relation").Default("other"),
		field.Bool("is_primary").Default(false),
		field.Bool("is_payer").Default(false),
		field.Bool("receives_messages").Default(true),
		field.String("note").Default(""),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (StudentContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("student", Student.Type).
			Ref("student_contacts").
			Field("student_id").
			Required().
			Unique(),

		edge.From("contact", Contact.Type).
			Ref("student_contacts").
			Field("contact_id").
			Required().
			Unique(),
	}
}
