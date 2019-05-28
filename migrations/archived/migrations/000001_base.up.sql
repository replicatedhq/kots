-- CREATE EXTENSION "pgcrypto";

-- CREATE TABLE github_users (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   username varchar(255) NOT NULL
-- );

-- CREATE TABLE bitbucket_users (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   username varchar(255) NOT NULL
-- );

-- CREATE TABLE gitlab_users (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   username varchar(255) NOT NULL
-- );

-- CREATE TABLE users (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   github_user_id uuid REFERENCES github_users,
--   bitbucket_user_id uuid REFERENCES bitbucket_users,
--   gitlab_user_id uuid REFERENCES gitlab_users
-- );

-- CREATE TABLE watches (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid()
-- );

-- CREATE TABLE users_watches (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   watch_id uuid REFERENCES watches,
--   user_id uuid REFERENCES users
-- );

-- CREATE TABLE sessions (
--   id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
--   user_id uuid REFERENCES users,
--   token varchar(255) NOT NULL,
--   expiry timestamp NOT NULL
-- );
