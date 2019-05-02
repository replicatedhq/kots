create table ship_unfork (
  id text not null primary key,
  upstream_uri text not null,
  fork_uri text not null,
  created_at timestamp without time zone not null,
  finished_at timestamp without time zone,
  result text,
  user_id text not null
);

