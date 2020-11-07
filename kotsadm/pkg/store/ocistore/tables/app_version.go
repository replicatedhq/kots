package tables

import (
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
)

func AppVersion() schemasv1alpha4.TableSpec {
	return schemasv1alpha4.TableSpec{
		Name: "app_version",
		Schema: &schemasv1alpha4.TableSchema{
			SQLite: &schemasv1alpha4.SqliteTableSchema{
				PrimaryKey: []string{
					"app_id",
					"sequence",
				},
				Columns: []*schemasv1alpha4.SqliteTableColumn{
					{
						Name: "app_id",
						Type: "text",
					},
					{
						Name: "sequence",
						Type: "int",
					},
					{
						Name: "update_cursor",
						Type: "text",
					},
					{
						Name: "channel_id",
						Type: "text",
					},
					{
						Name: "channel_name",
						Type: "text",
					},
					{
						Name: "upstream_released_at",
						Type: "timestamp",
					},
					{
						Name: "created_at",
						Type: "timestamp",
					},
					{
						Name: "version_label",
						Type: "text",
						Constraints: &schemasv1alpha4.SqliteTableColumnConstraints{
							NotNull: &trueValue,
						},
					},
					{
						Name: "release_notes",
						Type: "text",
					},
					{
						Name: "supportbundle_spec",
						Type: "text",
					},
					{
						Name: "preflight_spec",
						Type: "text",
					},
					{
						Name: "analyzer_spec",
						Type: "text",
					},
					{
						Name: "app_spec",
						Type: "text",
					},
					{
						Name: "kots_app_spec",
						Type: "text",
					},
					{
						Name: "kots_installation_spec",
						Type: "text",
					},
					{
						Name: "kots_license",
						Type: "text",
					},
					{
						Name: "config_spec",
						Type: "text",
					},
					{
						Name: "config_values",
						Type: "text",
					},
					{
						Name: "applied_at",
						Type: "timestamp",
					},
					{
						Name: "status",
						Type: "text",
					},
					{
						Name: "encryption_key",
						Type: "text",
					},
					{
						Name: "backup_spec",
						Type: "text",
					},
				},
			},
		},
	}
}
