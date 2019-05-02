create table pullrequest_history (
  notification_id text not null,
  pullrequest_number int not null,
  version_label text not null,
  org text not null,
  repo text not null,
  branch text,
  root_path text,
  created_at timestamp without time zone not null,
  primary key (notification_id, org, repo, pullrequest_number)
);
