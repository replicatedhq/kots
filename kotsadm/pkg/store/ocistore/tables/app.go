package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

var (
	defaultCron = "@default"
)

func App() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app",
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
						Name: "name",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "icon_uri",
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
						Name: "updated_at",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "slug",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
						Default: &gitopsString,
					},
					{
						Name: "upstream_uri",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "license",
						Type: "text",
					},
					{
						Name: "current_sequence",
						Type: "int",
					},
					{
						Name: "last_update_check_at",
						Type: "text",
					},
					{
						Name: "is_app_users",
						Type: "bool",
					},
					{
						Name: "registry_hostname",
						Type: "text",
					},
					{
						Name: "registry_username",
						Type: "text",
					},
					{
						Name: "registry_password",
						Type: "text",
					},
					{
						Name: "registry_password_enc",
						Type: "text",
					},
					{
						Name: "namespace",
						Type: "text",
					},
					{
						Name: "last_registry_sync",
						Type: "timestamp",
					},
					{
						Name: "install_state",
						Type: "text",
					},
					{
						Name:    "is_airgap",
						Type:    "bool",
						Default: &falseValueString,
					},
					{
						Name: "snapshot_ttl_new",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
						Default: &thirtyDaysInHours,
					},
					{
						Name: "snapshot_schedule",
						Type: "text",
					},
					{
						Name: "restore_in_progress_name",
						Type: "text",
					},
					{
						Name: "restore_undeploy_status",
						Type: "text",
					},
					{
						Name:    "update_checker_spec",
						Type:    "text",
						Default: &defaultCron,
					},
				},
			},
		},
	}
}
