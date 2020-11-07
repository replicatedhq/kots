package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func SupportBundle() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "supportbundle",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "id",
						Type: "text",
					},
					{
						Name: "slug",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "watch_id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "name",
						Type: "text",
					},
					{
						Name: "size",
						Type: "int",
					},
					{
						Name: "status",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "tree_index",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "analysis_id",
						Type: "text",
					},
					{
						Name: "created_at",
						Type: "timestamp",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "uploaded_at",
						Type: "timestamp",
					},
					{
						Name: "is_archived",
						Type: "bool",
					},
					{
						Name: "redact_report",
						Type: "text",
					},
				},
			},
		},
	}
}
