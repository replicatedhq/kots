create table cluster (
  id text primary key not null,
  title text not null,
  slug text not null unique,
  created_at timestamp without time zone not null,
  updated_at timestamp without time zone
);

create table user_cluster (
  user_id text not null,
  cluster_id text not null
);
