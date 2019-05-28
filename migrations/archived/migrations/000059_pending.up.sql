create table ship_init_pending (
  id text primary key not null,
  upstream_uri text not null,
  requested_upstream_uri text not null,
  title text not null,
  created_at timestamp without time zone not null,
  finished_at timestamp without time zone,
  result text
);

create table ship_init_pending_user (
  user_id text not null,
  ship_init_pending_id text not null
);

