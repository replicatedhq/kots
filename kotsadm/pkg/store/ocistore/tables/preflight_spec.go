package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func PreflightSpec() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "preflight_spec",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"watch_id",
					"sequence",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "watch_id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "sequence",
						Type: "int",
					},
					{
						Name: "spec",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
				},
			},
		},
	}
}
