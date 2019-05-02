create table image_watch_batch (
  id text primary key not null,
  user_id text null,
  images_input text not null,
  created_at timestamp without time zone not null
);

create table image_watch (
  id text primary key not null,
  batch_id text not null,
  image_name text not null,
  checked_at timestamp without time zone null,
  is_private boolean not null default false,
  versions_behind int not null default 0,
  detected_version text null,
  latest_version text null,
  compatible_version text null,
  check_error text null
);