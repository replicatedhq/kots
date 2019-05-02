
drop table worker_init_session;

create table ship_init (
  id text primary key not null,
  upstream_uri text not null,
  created_at timestamp without time zone not null,
  finished_at timestamp without time zone,
  result text
);