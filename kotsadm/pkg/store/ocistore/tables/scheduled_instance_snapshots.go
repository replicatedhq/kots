package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func ScheduledInstanceSnapshots() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "scheduled_instance_snapshots",
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
						Name: "cluster_id",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "scheduled_timestmap",
						Type: "timestamp",
					},
					{
						Name: "backup_name",
						Type: "text",
					},
				},
			},
		},
	}
}
