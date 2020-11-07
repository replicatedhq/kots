package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func APITaskStatus() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "api_task_status",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"id",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "updated_at",
						Type: "timestamp",
					},
					{
						Name: "current_message",
						Type: "text",
					},
					{
						Name: "status",
						Type: "text",
					},
				},
			},
		},
	}
}
