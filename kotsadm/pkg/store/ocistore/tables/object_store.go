package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func ObjectStore() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "object_store",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"filepath",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "filepath",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "emcoded_block",
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
