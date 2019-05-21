
create table ship_output_files (
  watch_id text not null,
  created_at timestamp without time zone not null,
  sequence integer default 0,
  filepath text not null,
  primary key(watch_id, sequence)
);
