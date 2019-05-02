create table if not exists feature (
  id text not null primary key,
  created_at timestamp without time zone
);

create table if not exists user_feature (
  user_id text not null,
  feature_id text not null,
  primary key (user_id, feature_id)
);

create table if not exists watch_feature (
  watch_id text not null,
  feature_id text not null,
  primary key (watch_id, feature_id)
);

insert into feature (id, created_at) values ('flux-integration', now()) on conflict do nothing;
