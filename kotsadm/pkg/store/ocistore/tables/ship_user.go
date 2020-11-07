package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func ShipUser() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "ship_user",
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
						Name: "created_at",
						Type: "timestamp",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "github_id",
						Type: "int",
					},
					{
						Name: "last_login",
						Type: "timestamp",
					},
				},
			},
		},
	}
}
