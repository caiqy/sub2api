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

// UsageLogDetail 定义 usage log 对应的明细快照。
type UsageLogDetail struct {
	ent.Schema
}

func (UsageLogDetail) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_log_details"},
	}
}

func (UsageLogDetail) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("usage_log_id").
			Unique(),
		field.Text("request_headers").
			Default(""),
		field.Text("request_body").
			Default(""),
		field.Text("response_headers").
			Default(""),
		field.Text("response_body").
			Default(""),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UsageLogDetail) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_log", UsageLog.Type).
			Ref("detail").
			Field("usage_log_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (UsageLogDetail) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
	}
}
