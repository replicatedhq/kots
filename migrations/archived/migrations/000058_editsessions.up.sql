create table ship_edit (
  id text primary key not null,
  watch_id text not null,
  user_id text not null,
  result text,
  created_at timestamp without time zone not null,
  finished_at timestamp without time zone
);
