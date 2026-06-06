package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuditLog struct{ ent.Schema }

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("actor_user_id").Optional().Nillable(),
		field.String("actor_label").Default("system"),
		field.String("entity_type"),
		field.Int("entity_id").Optional().Nillable(),
		field.String("action"),
		field.String("summary"),
		field.String("before_json").Default(""),
		field.String("after_json").Default(""),
		field.Int("student_id").Optional().Nillable(),
		field.Int("invoice_id").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
	}
}

func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("actor_user", User.Type).
			Ref("audit_logs").
			Unique().
			Field("actor_user_id"),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("actor_user_id", "created_at"),
		index.Fields("entity_type", "entity_id"),
		index.Fields("action", "created_at"),
		index.Fields("student_id", "created_at"),
		index.Fields("invoice_id", "created_at"),
	}
}
