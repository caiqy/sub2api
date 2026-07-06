package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserResourceOverride stores per-user resource allow/deny overrides.
type UserResourceOverride struct {
	ent.Schema
}

func (UserResourceOverride) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_resource_overrides"},
	}
}

func (UserResourceOverride) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("resource_type").MaxLen(50),
		field.Int64("resource_id"),
		field.String("effect").MaxLen(20),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserResourceOverride) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
	}
}

func (UserResourceOverride) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "resource_type", "resource_id", "effect").Unique(),
		index.Fields("user_id", "resource_type", "effect"),
	}
}
