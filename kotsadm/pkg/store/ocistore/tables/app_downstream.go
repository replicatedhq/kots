package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func AppDownstream() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app_downstream",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"app_id",
					"cluster_id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "app_id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "cluster_id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "downstream_name",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "current_sequence",
						Type: "int",
					},
				},
			},
		},
	}
}
