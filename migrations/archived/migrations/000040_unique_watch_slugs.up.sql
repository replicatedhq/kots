-- append '-dup' to latest created entry of each set of duplicates
-- this relies on timestamps being unique, but I am confident that they are
-- it also relies on there not being more than two entries with the same slug
UPDATE watch
SET slug = slug || '-dup'
WHERE created_at IN (
  SELECT MAX(created_at)
  FROM watch
  GROUP BY slug
  HAVING ( COUNT(slug) > 1)
);

-- add a 'unique' constraint to the table to ensure future slugs are not duplicates
ALTER TABLE watch ADD CONSTRAINT UQ_watch_slug UNIQUE (slug);
