package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// QuotaPoolGroupLimit stores an access group's percent limit within a quota pool.
type QuotaPoolGroupLimit struct {
	ent.Schema
}

func (QuotaPoolGroupLimit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "quota_pool_group_limits"},
	}
}

func (QuotaPoolGroupLimit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (QuotaPoolGroupLimit) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("quota_pool_id"),
		field.Int64("group_id"),
		field.Int("percent_basis_points").
			Default(0),
		field.Bool("enabled").
			Default(true),
	}
}

func (QuotaPoolGroupLimit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("quota_pool", QuotaPool.Type).
			Ref("group_limits").
			Field("quota_pool_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("quota_pool_limits").
			Field("group_id").
			Unique().
			Required(),
	}
}

func (QuotaPoolGroupLimit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("quota_pool_id", "group_id").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("group_id"),
		index.Fields("enabled"),
	}
}
