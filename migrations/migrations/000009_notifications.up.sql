create table ship_notification (
  id text not null primary key,
  watch_id text not null,
  created_at timestamp without time zone not null,
  updated_at timestamp without time zone,
  triggered_at timestamp without time zone
);

create table webhook_notification (
  notification_id text not null,
  destination_uri text not null,
  created_at timestamp without time zone not null
);

create table email_notification (
  notification_id text not null,
  recipient text not null,
  created_at timestamp without time zone not null
);

create table pullrequest_notification (
  notification_id text not null,
  org text not null,
  repo text not null,
  branch text,
  root_path text,
  created_at timestamp without time zone not null
);