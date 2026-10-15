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

// QuotaPoolAccount binds an upstream account to a shared quota pool.
type QuotaPoolAccount struct {
	ent.Schema
}

func (QuotaPoolAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "quota_pool_accounts"},
	}
}

func (QuotaPoolAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (QuotaPoolAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("quota_pool_id"),
		field.String("platform").
			MaxLen(50).
			NotEmpty(),
		field.Int64("account_id"),
		field.Bool("enabled").
			Default(true),
		field.Float("last_usage_percent").
			Default(0),
	}
}

func (QuotaPoolAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("quota_pool", QuotaPool.Type).
			Ref("accounts").
			Field("quota_pool_id").
			Unique().
			Required(),
		edge.From("account", Account.Type).
			Ref("quota_pool_accounts").
			Field("account_id").
			Unique().
			Required(),
		edge.To("group_usages", QuotaPoolGroupUsage.Type),
	}
}

func (QuotaPoolAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("quota_pool_id", "platform", "account_id").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("platform", "account_id"),
		index.Fields("enabled"),
	}
}
