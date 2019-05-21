ALTER TABLE watch ALTER COLUMN title SET NOT NULL;
UPDATE watch SET title = 'default' WHERE title = '';
ALTER TABLE watch ADD CONSTRAINT non_empty_title CHECK(length(title)>0);
