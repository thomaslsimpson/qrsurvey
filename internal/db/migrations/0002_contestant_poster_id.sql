-- Attribution for entries made through the direct-entry (AMOE) link, which
-- skips the survey entirely and so produces zero answer rows — without this
-- column there would be no record of which poster's alternate-entry link
-- was used. Survey-completing entries get this filled in too, for
-- consistency, even though it's also derivable via their answer rows.
ALTER TABLE contestant ADD COLUMN poster_id INTEGER REFERENCES poster(id);
CREATE INDEX idx_contestant_poster_id ON contestant(poster_id);
