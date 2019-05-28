-- there might be data, so defaulting this to '';
alter table pullrequest_notification add column github_installation_id text not null default '';