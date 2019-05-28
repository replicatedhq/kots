ALTER TABLE pending_pullrequest_notification ADD COLUMN pullrequest_number int default -1;
ALTER TABLE pending_pullrequest_notification ADD COLUMN watch_id text;