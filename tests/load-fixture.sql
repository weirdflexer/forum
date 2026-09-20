-- Apply only in a separate database ending in _load_test, after migrations.
-- Creates 10,000 topics and 50,000 posts in one transaction. Repeat is a no-op.
BEGIN;
DO $$ BEGIN
 IF current_database() NOT LIKE '%\_load\_test' ESCAPE '\' THEN RAISE EXCEPTION 'Use dedicated *_load_test database'; END IF;
END $$;
WITH section AS (
 INSERT INTO sections(slug,title,description) VALUES('load-test','Нагрузочный стенд','Синтетические данные')
 ON CONFLICT(slug) DO NOTHING RETURNING id
), created AS (
 INSERT INTO topics(section_id,title)
 SELECT section.id, 'Нагрузочная тема '||n FROM section CROSS JOIN generate_series(1,10000) AS n RETURNING id
)
INSERT INTO posts(topic_id,number,body)
SELECT created.id,n,'Нагрузочное сообщение '||n FROM created CROSS JOIN generate_series(1,5) AS n;
COMMIT;
SELECT id AS example_topic_id FROM topics WHERE section_id=(SELECT id FROM sections WHERE slug='load-test') LIMIT 1;
