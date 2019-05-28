
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO shipcloud_rw;

-- --grant privileges to all future tables created by shipcloud in the public schema as well
-- ALTER DEFAULT PRIVILEGES
--   FOR ROLE shipcloud
--     IN SCHEMA public
--     GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO shipcloud_rw
