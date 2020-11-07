package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func UserCluster() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "user_cluster",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "user_id",
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
				},
			},
		},
	}
}
