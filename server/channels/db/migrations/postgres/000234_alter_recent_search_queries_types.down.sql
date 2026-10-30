DROP INDEX IF EXISTS idx_recentsearchqueries_userid_termsmd5_searchtype_teamid;
CREATE UNIQUE INDEX IF NOT EXISTS idx_recentsearchqueries_userid_terms_searchtype_teamid
    ON recentsearchqueries (userid, terms, searchtype, teamid);

ALTER TABLE recentsearchqueries
    ALTER COLUMN id TYPE CHAR(26),
    ALTER COLUMN userid TYPE CHAR(26),
    ALTER COLUMN teamid TYPE CHAR(26),
    ALTER COLUMN teamid SET DEFAULT '';
