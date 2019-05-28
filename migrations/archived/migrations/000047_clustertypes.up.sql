create table cluster_github (
  cluster_id text not null primary key,
  owner text not null,
  repo text not null,
  branch text null,
  installation_id integer not null
);

alter table cluster add column cluster_type text not null default 'gitops';
alter table cluster alter column token drop not null;

alter table ship_init add column cluster_id text null;
alter table ship_init add column github_path text null;
