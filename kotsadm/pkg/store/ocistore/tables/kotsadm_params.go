package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func KotsadmParams() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "kotsadm_params",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"key",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "key",
						Type: "text",
					},
					{
						Name: "value",
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
