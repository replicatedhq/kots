alter table cluster add column token text null unique;
update cluster set token = md5(random()::text) where token is null;
alter table cluster alter column token set not null;

create table watch_cluster (
  watch_id text not null,
  cluster_id text not null
);
