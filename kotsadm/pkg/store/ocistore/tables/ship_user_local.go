package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func ShipUserLocal() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "ship_user_local",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"user_id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "user_id",
						Type: "text",
					},
					{
						Name: "password_bcrypt",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "first_name",
						Type: "text",
					},
					{
						Name: "last_name",
						Type: "text",
					},
					{
						Name: "email",
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
