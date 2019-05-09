create table ship_user_local (
  user_id text not null primary key,
  password_bcrypt text not null,
  first_name text null,
  last_name text null
);

alter table github_user add column user_id text not null;

update github_user
    set user_id = ship_user.id
    from ship_user
    WHERE github_user.github_id = ship_user.github_id;

alter table ship_user drop column github_id;
