
create table worker_init_session (
  id text primary key not null,
  watch_id text not null,
  current_status text not null,
  upstream_uri text not null,
  created_at timestamp without time zone not null,
  updated_at timestamp without time zone
);

create table worker_watch_state (
  id text primary key not null,
  watch_id text not null,
  current_status text not null,
  created_at timestamp without time zone not null,
  updated_at timestamp without time zone
);
create unique index worker_watch_state_watch_id_idx on worker_watch_state (watch_id);
