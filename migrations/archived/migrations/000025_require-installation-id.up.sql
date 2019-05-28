delete from pullrequest_notification where github_installation_id = '';
alter table pullrequest_notification alter column github_installation_id drop default;
alter table pullrequest_notification
    alter column github_installation_id
    type int using (github_installation_id::integer);
alter table pullrequest_notification alter column github_installation_id set not null;