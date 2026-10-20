package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// ReleaseNoticeView keeps the account's original 24-hour window across devices.
type ReleaseNoticeView struct{ ent.Schema }

func (ReleaseNoticeView) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.String("release_id"),
		field.Time("first_seen_at").Default(time.Now).Immutable(),
		field.Time("dismissed_at").Optional().Nillable(),
	}
}
func (ReleaseNoticeView) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("release_notice_views").Field("user_id").Unique().Required(),
	}
}
func (ReleaseNoticeView) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "release_id").Unique()}
}
