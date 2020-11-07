package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func AppStatus() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app_status",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"app_id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "app_id",
						Type: "text",
					},
					{
						Name: "resource_states",
						Type: "text",
					},
					{
						Name: "updated_at",
						Type: "timestamp",
					},
				},
			},
		},
	}
}
