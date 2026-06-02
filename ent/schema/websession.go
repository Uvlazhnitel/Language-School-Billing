package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type WebSession struct{ ent.Schema }

func (WebSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_hash").Unique(),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now),
		field.Time("last_seen_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WebSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Unique().Required(),
	}
}
