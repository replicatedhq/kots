package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func AppDownstreamVersion() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app_downstream_version",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"app_id",
					"cluster_id",
					"sequence",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "app_id",
						Type: "text",
					},
					{
						Name: "cluster_id",
						Type: "text",
					},
					{
						Name: "sequence",
						Type: "int",
					},
					{
						Name: "parent_sequence",
						Type: "int",
					},
					{
						Name: "created_at",
						Type: "timestamp",
					},
					{
						Name: "applied_at",
						Type: "timestamp",
					},
					{
						Name: "version_label",
						Type: "timestamp",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "status",
						Type: "text",
					},
					{
						Name: "status_info",
						Type: "text",
					},
					{
						Name: "source",
						Type: "text",
					},
					{
						Name: "diff_summary",
						Type: "text",
					},
					{
						Name: "diff_summary_error",
						Type: "text",
					},
					{
						Name: "preflight_result_created_at",
						Type: "timestamp",
					},
					{
						Name:    "preflight_ignore_permissions",
						Type:    "bool",
						Default: &falseValueString,
					},
					{
						Name: "git_commit_url",
						Type: "text",
					},
					{
						Name:    "git_deployable",
						Type:    "bool",
						Default: &trueValueString,
					},
				},
			},
		},
	}
}
