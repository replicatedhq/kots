create table watch_version (
  watch_id text not null,
  created_at timestamp without time zone,
  version_label text not null,
  status text not null default 'unknown',
  source_branch text null,
  sequence integer default 0,
  pullrequest_number integer null
);

