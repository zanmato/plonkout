DROP TABLE food_aliases;
ALTER TABLE foods DROP COLUMN search, DROP COLUMN search_terms;
ALTER TABLE lmv.releases DROP COLUMN vocabulary;
ALTER TABLE lmv.foods DROP COLUMN search, DROP COLUMN search_terms;
