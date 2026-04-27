package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Teacher struct{ ent.Schema }

func (Teacher) Fields() []ent.Field {
	return []ent.Field{
		field.String("full_name"),
		field.Bool("is_active").Default(true),
	}
}

func (Teacher) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("courses", Course.Type),
	}
}
