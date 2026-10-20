package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type User struct{ ent.Schema }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").StorageKey("email").Unique(),
		field.String("password_hash"),
		field.String("role").Default("admin"),
		field.Bool("is_active").Default(true),
		field.String("ui_locale").Default("lv-LV"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sessions", WebSession.Type),
		edge.To("release_notice_views", ReleaseNoticeView.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("audit_logs", AuditLog.Type),
	}
}
