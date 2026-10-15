package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// QuotaPool defines a shared account quota pool.
type QuotaPool struct {
	ent.Schema
}

func (QuotaPool) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "quota_pools"},
	}
}

func (QuotaPool) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (QuotaPool) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Bool("enabled").
			Default(true),
		field.String("period_type").
			MaxLen(20).
			Default("current"),
		field.String("default_action").
			MaxLen(20).
			Default("allow"),
	}
}

func (QuotaPool) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("accounts", QuotaPoolAccount.Type),
		edge.To("group_limits", QuotaPoolGroupLimit.Type),
		edge.To("group_usages", QuotaPoolGroupUsage.Type),
	}
}

func (QuotaPool) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("enabled"),
		index.Fields("period_type"),
	}
}
