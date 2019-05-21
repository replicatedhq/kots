create table pending_pullrequest_notification (
  pullrequest_history_id text not null,
  org text not null,
  repo text not null,
  branch text not null,
  root_path text,
  created_at timestamp without time zone not null,
  github_installation_id integer not null
)
