package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

var (
	gitopsString      = "gitops"
	thirtyDaysInHours = "720h"
	falseValueString  = "false"
	trueValueString   = "true"
)

func Cluster() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "cluster",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"id",
				},
				Indexes: []*schemasv1alpha4.SqliteTableIndex{
					{
						Columns: []string{
							"token",
						},
						Name:     "cluster_token_key",
						IsUnique: true,
					},
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "title",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "slug",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "created_at",
						Type: "timestamp",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "updated_at",
						Type: "text",
					},
					{
						Name: "token",
						Type: "text",
					},
					{
						Name: "cluster_type",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
						Default: &gitopsString,
					},
					{
						Name: "is_all_users",
						Type: "bool",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
						Default: &falseValueString,
					},
					{
						Name: "snapshot_schedule",
						Type: "text",
					},
					{
						Name: "snapshot_ttl",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
						Default: &thirtyDaysInHours,
					},
				},
			},
		},
	}
}
