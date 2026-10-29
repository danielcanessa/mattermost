-- CHAR(26) pads an empty teamid to 26 spaces on read; VARCHAR(26) keeps it empty.
ALTER TABLE recentsearchqueries
    ALTER COLUMN id TYPE VARCHAR(26),
    ALTER COLUMN userid TYPE VARCHAR(26),
    ALTER COLUMN teamid TYPE VARCHAR(26),
    ALTER COLUMN teamid SET DEFAULT '';

UPDATE recentsearchqueries SET teamid = btrim(teamid) WHERE teamid <> btrim(teamid);

-- A btree entry over the full terms (up to 2048 multibyte runes) can exceed the
-- ~2.7 KB index row limit, so the unique key uses md5(terms) instead.
DROP INDEX IF EXISTS idx_recentsearchqueries_userid_terms_searchtype_teamid;
CREATE UNIQUE INDEX IF NOT EXISTS idx_recentsearchqueries_userid_termsmd5_searchtype_teamid
    ON recentsearchqueries (userid, md5(terms), searchtype, teamid);
