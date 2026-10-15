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

// QuotaPoolGroupUsage stores accumulated group usage in a quota pool account window.
type QuotaPoolGroupUsage struct {
	ent.Schema
}

func (QuotaPoolGroupUsage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "quota_pool_group_usage"},
	}
}

func (QuotaPoolGroupUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (QuotaPoolGroupUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("quota_pool_id"),
		field.Int64("quota_pool_account_id"),
		field.Int64("group_id"),
		field.String("period_type").
			MaxLen(20).
			Default("current"),
		field.String("period_key").
			MaxLen(20).
			Default("current"),
		field.Float("used_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
	}
}

func (QuotaPoolGroupUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("quota_pool", QuotaPool.Type).
			Ref("group_usages").
			Field("quota_pool_id").
			Unique().
			Required(),
		edge.From("quota_pool_account", QuotaPoolAccount.Type).
			Ref("group_usages").
			Field("quota_pool_account_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("quota_pool_usages").
			Field("group_id").
			Unique().
			Required(),
	}
}

func (QuotaPoolGroupUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("quota_pool_id", "quota_pool_account_id", "group_id", "period_type", "period_key").
			Unique(),
		index.Fields("quota_pool_id", "group_id", "period_type", "period_key"),
		index.Fields("quota_pool_account_id"),
		index.Fields("group_id"),
		index.Fields("period_type", "period_key"),
	}
}
