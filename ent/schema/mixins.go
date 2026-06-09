package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type VersionMixin struct{ mixin.Schema }

func (VersionMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int("version").Default(1),
	}
}
