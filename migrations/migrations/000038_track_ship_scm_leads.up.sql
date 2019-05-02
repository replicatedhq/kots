create table track_scm_leads (
  id text primary key not null,
  deployment_type text not null,
  email_address text not null,
  scm_provider text not null,
  created_at timestamp without time zone not null,
  followed_up boolean not null default false
);