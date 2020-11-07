package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func UserApp() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "user_app",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"user_id",
					"app_id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "user_id",
						Type: "text",
					},
					{
						Name: "app_id",
						Type: "text",
					},
				},
			},
		},
	}
}
