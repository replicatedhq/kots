package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func AppDownstreamOutput() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app_downstream_output",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"app_id",
					"cluster_id",
					"downstream_sequence",
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
						Name: "downstream_sequence",
						Type: "int",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "dryrun_stdout",
						Type: "text",
					},
					{
						Name: "dryrun_stderr",
						Type: "text",
					},
					{
						Name: "apply_stdout",
						Type: "text",
					},
					{
						Name: "apply_stderr",
						Type: "text",
					},
					{
						Name: "is_error",
						Type: "bool",
					},
				},
			},
		},
	}
}
