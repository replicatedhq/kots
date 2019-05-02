drop table if exists sessions;
create table session (
  id text primary key not null,
  user_id text not null,
  metadata text not null,
  expire_at timestamp without time zone not null
);

drop table if exists users cascade;
create table ship_user (
  id text not null primary key,
  github_id int null,
  created_at timestamp without time zone
);

drop table if exists bitbucket_users;

drop table if exists gitlab_users;

drop table if exists github_users;
create table github_user (
  username text not null primary key,
  github_id int not null
);

drop table if exists watches cascade;
create table watch (
  id text not null primary key,
  current_state text,
  title text,
  icon_uri text,
  created_at timestamp without time zone not null,
  updated_at timestamp without time zone,
  deployed_at timestamp without time zone
);

drop table if exists users_watches;
create table user_watch (
  user_id text not null,
  watch_id text not null,
  primary key (user_id, watch_id)
);

