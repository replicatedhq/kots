package persistence

var tables = []string{
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: api-task-status
spec:
  database: kotsadm
  name: api_task_status
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: updated_at
        type: integer
      - name: current_message
        type: text
      - name: status
        type: text
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app-downstream-output
spec:
  database: kotsadm
  name: app_downstream_output
  requires: []
  schema:
    sqlite:
      primaryKey:
        - app_id
        - cluster_id
        - downstream_sequence
      columns:
      - name: app_id
        type: text
        constraints:
          notNull: true
      - name: cluster_id
        type: text
        constraints:
          notNull: true
      - name: downstream_sequence
        type: integer
        constraints:
          notNull: true
      - name: dryrun_stdout
        type: text
      - name: dryrun_stderr
        type: text
      - name: apply_stdout
        type: text
      - name: apply_stderr
        type: text
      - name: helm_stdout
        type: text
      - name: helm_stderr
        type: text
      - name: is_error
        type: boolean`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app-downstream-version
spec:
  database: kotsadm
  name: app_downstream_version
  requires: []
  schema:
    sqlite:
      primaryKey:
        - app_id
        - cluster_id
        - sequence
      columns:
      - name: app_id
        type: text
      - name: cluster_id
        type: text
      - name: sequence
        type: integer
      - name: parent_sequence
        type: integer
      - name: created_at
        type: integer
      - name: applied_at
        type: integer
      - name: version_label
        type: text
        constraints:
          notNull: true
      - name: status
        type: text
      - name: status_info
        type: text
      - name: source
        type: text
      - name: diff_summary
        type: text
      - name: diff_summary_error
        type: text
      - name: preflight_progress
        type: text
      - name: preflight_result
        type: text
      - name: preflight_result_created_at
        type: integer
      - name: preflight_ignore_permissions
        type: boolean
        default: "false"
      - name: preflight_skipped
        type: boolean
        default: "false"
      - name: git_commit_url
        type: text
      - name: git_deployable
        type: boolean
        default: "true"
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app-downstream
spec:
  database: kotsadm
  name: app_downstream
  requires: []
  schema:
    sqlite:
      primaryKey:
        - app_id
        - cluster_id
      columns:
      - name: app_id
        type: text
        constraints:
          notNull: true
      - name: cluster_id
        type: text
        constraints:
          notNull: true
      - name: downstream_name
        type: text
        constraints:
          notNull: true
      - name: current_sequence
        type: integer
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app-status
spec:
  database: kotsadm
  name: app_status
  requires: []
  schema:
    sqlite:
      primaryKey:
        - app_id
      columns:
      - name: app_id
        type: text
      - name: resource_states
        type: text
      - name: updated_at
        type: integer
      - name: sequence
        type: integer
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app-version
spec:
  database: kotsadm
  name: app_version
  requires: []
  schema:
    sqlite:
      primaryKey:
        - app_id
        - sequence
      columns:
      - name: app_id
        type: text
      - name: sequence
        type: integer
      - name: update_cursor
        type: text
      - name: channel_id
        type: text
      - name: channel_name
        type: text
      - name: upstream_released_at
        type: integer
      - name: created_at
        type: integer
      - name: version_label
        type: text
        constraints:
          notNull: true
      - name: release_notes
        type: text
      - name: supportbundle_spec
        type: text
      - name: preflight_spec
        type: text
      - name: analyzer_spec
        type: text
      - name: app_spec
        type: text
      - name: kots_app_spec
        type: text
      - name: kots_installation_spec
        type: text
      - name: kots_license
        type: text
      - name: config_spec
        type: text
      - name: config_values
        type: text
      - name: applied_at
        type: integer
      - name: status
        type: text
      - name: encryption_key
        type: text
      - name: backup_spec
        type: text
      - name: identity_spec
        type: text
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: app
spec:
  database: kotsadm
  name: app
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: name
        type: text
        constraints:
          notNull: true
      - name: icon_uri
        type: text
      - name: created_at
        type: integer
        constraints:
          notNull: true
      - name: updated_at
        type: integer
      - name: slug
        type: text
        constraints:
          notNull: true
      - name: upstream_uri
        type: text
        constraints:
          notNull: true
      - name: license
        type: text
      - name: current_sequence
        type: integer
      - name: last_update_check_at
        type: integer
      - name: is_all_users
        type: boolean
      - name: registry_hostname
        type: text
      - name: registry_username
        type: text
      - name: registry_password
        type: text
      - name: registry_password_enc
        type: text
      - name: namespace
        type: text
      - name: registry_is_readonly
        type: boolean
      - name: last_registry_sync
        type: integer
      - name: install_state
        type: text
      - name: is_airgap
        type: boolean
        default: "false"
      - name: snapshot_ttl_new
        type: text
        default: '720h'
        constraints:
          notNull: true
      - name: snapshot_schedule
        type: text
      - name: restore_in_progress_name
        type: text
      - name: restore_undeploy_status
        type: text
      - name: update_checker_spec
        type: text
        default: '@default'
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: cluster
spec:
  database: kotsadm
  name: cluster
  schema:
    sqlite:
      primaryKey:
      - id
      indexes:
      - columns:
        - token
        name: cluster_token_key
        isUnique: true
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: title
        type: text
        constraints:
          notNull: true
      - name: slug
        type: text
        constraints:
          notNull: true
      - name: created_at
        type: integer
        constraints:
          notNull: true
      - name: updated_at
        type: integer
      - name: token
        type: text
      - name: cluster_type
        type: text
        constraints:
          notNull: true
        default: 'gitops'
      - name: is_all_users
        type: boolean
        constraints:
          notNull: true
        default: "false"
      - name: snapshot_schedule
        type: text
      - name: snapshot_ttl
        type: text
        default: '720h'
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kotsadm-params
spec:
  database: kotsadm
  name: kotsadm_params
  requires: []
  schema:
    sqlite:
      primaryKey:
      - key
      columns:
      - name: key
        type: text
      - name: value
        type: text
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: object-store
spec:
  database: kotsadm
  name: object_store
  requires: []
  schema:
    sqlite:
      primaryKey:
      - filepath
      columns:
      - name: filepath
        type: text
        constraints:
          notNull: true
      - name: encoded_block
        type: text
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: pending-supportbundle
spec:
  database: kotsadm
  name: pending_supportbundle
  requires: []
  schema:
    sqlite:
      primaryKey:
        - id
      columns:
      - name: id
        type: text
      - name: app_id
        type: text
        constraints:
          notNull: true
      - name: cluster_id
        type: text
        constraints:
          notNull: true
      - name: created_at
        type: integer
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: preflight-result
spec:
  database: kotsadm
  name: preflight_result
  requires: []
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: watch_id
        type: text
        constraints:
          notNull: true
      - name: result
        type: text
        constraints:
          notNull: true
      - name: created_at
        type: integer
        constraints:
          notNull: true

`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: preflight-spec
spec:
  database: kotsadm
  name: preflight_spec
  schema:
    sqlite:
      primaryKey:
      - watch_id
      - sequence
      columns:
      - name: watch_id
        type: text
        constraints:
          notNull: true
      - name: sequence
        type: int
        constraints:
          notNull: true
      - name: spec
        type: text
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: scheduled-instance-snapshots
spec:
  database: kotsadm
  name: scheduled_instance_snapshots
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: cluster_id
        type: text
        constraints:
          notNull: true
      - name: scheduled_timestamp
        type: integer
        constraints:
          notNull: true
      - name: backup_name
        type: text
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  name: scheduled-snapshots
spec:
  database: kotsadm
  name: scheduled_snapshots
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: app_id
        type: text
        constraints:
          notNull: true
      - name: scheduled_timestamp
        type: integer
        constraints:
          notNull: true
      - name: backup_name
        type: text
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: session
spec:
  database: kotsadm
  name: session
  requires: []
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: user_id
        type: text
        constraints:
          notNull: true
      - name: metadata
        type: text
        constraints:
          notNull: true
      - name: issued_at
        type: integer
      - name: expire_at
        type: integer
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: ship-user-local
spec:
  database: kotsadm
  name: ship_user_local
  requires: []
  schema:
    sqlite:
      indexes:
        - columns: [email]
          isUnique: true
      primaryKey:
      - user_id
      columns:
      - name: user_id
        type: text
        constraints:
          notNull: true
      - name: password_bcrypt
        type: text
        constraints:
          notNull: true
      - name: first_name
        type: text
      - name: last_name
        type: text
      - name: email
        type: text
        constraints:
          notNull: true
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: ship-user
spec:
  database: kotsadm
  name: ship_user
  requires: []
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: created_at
        type: integer
      - name: github_id
        type: integer
      - name: last_login
        type: integer
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: supportbundle-analysis
spec:
  database: kotsadm
  name: supportbundle_analysis
  requires: []
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: supportbundle_id
        type: text
        constraints:
          notNull: true
      - name: error
        type: text
      - name: max_severity
        type: text
      - name: insights
        type: text
      - name: created_at
        type: integer
        constraints:
          notNull: true
`,
	`## no longer used, must keep for migrations to complete
apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: supportbundle
spec:
  database: kotsadm
  name: supportbundle
  requires: []
  schema:
    sqlite:
      primaryKey:
      - id
      columns:
      - name: id
        type: text
        constraints:
          notNull: true
      - name: slug # TODO: unique????
        type: text
        constraints:
          notNull: true
      - name: watch_id
        type: text
        constraints:
          notNull: true
      - name: name
        type: text
      - name: size
        type: integer
      - name: status
        type: text
        constraints:
          notNull: true
      - name: tree_index
        type: text
      - name: analysis_id
        type: text
      - name: created_at
        type: integer
        constraints:
          notNull: true
      - name: uploaded_at
        type: integer
      - name: is_archived
        type: boolean
      - name: redact_report
        type: text
`,
	`
apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: user-app
spec:
  database: kotsadm
  name: user_app
  requires: []
  schema:
    sqlite:
      primaryKey:
      - user_id
      - app_id
      columns:
      - name: user_id
        type: text
      - name: app_id
        type: text
`,
	`apiVersion: schemas.schemahero.io/v1alpha4
kind: Table
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: user-cluster
spec:
  database: kotsadm
  name: user_cluster
  requires: []
  schema:
    sqlite:
      primaryKey: []
      columns:
      - name: user_id
        type: text
        constraints:
          notNull: true
      - name: cluster_id
        type: text
        constraints:
          notNull: true
`,
}
