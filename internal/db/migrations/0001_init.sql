CREATE TABLE survey (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  description TEXT NOT NULL,
  created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE contest (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  survey_id  INTEGER NOT NULL REFERENCES survey(id) ON DELETE RESTRICT,
  end_date   TEXT NOT NULL,
  prize      TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_contest_survey_id ON contest(survey_id);

CREATE TABLE poster (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  contest_id           INTEGER NOT NULL REFERENCES contest(id) ON DELETE RESTRICT,
  internal_poster_info TEXT NOT NULL,
  created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_poster_contest_id ON poster(contest_id);

CREATE TABLE survey_item (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  survey_id  INTEGER NOT NULL REFERENCES survey(id) ON DELETE CASCADE,
  question   TEXT NOT NULL,
  response_1 TEXT NOT NULL,
  response_2 TEXT NOT NULL,
  response_3 TEXT NOT NULL,
  response_4 TEXT NOT NULL,
  response_5 TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_survey_item_survey_id ON survey_item(survey_id);

-- name/phone/address per issue #2's data model; email added to match the
-- approved design's contest-entry screen (name, email, mobile). address
-- stays nullable and is not collected by the public form today.
CREATE TABLE contestant (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  contest_id INTEGER NOT NULL REFERENCES contest(id) ON DELETE RESTRICT,
  name       TEXT NOT NULL,
  email      TEXT,
  phone      TEXT NOT NULL,
  address    TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_contestant_contest_id ON contestant(contest_id);
CREATE UNIQUE INDEX uq_contestant_contest_phone ON contestant(contest_id, phone);

CREATE TABLE answer (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  contestant_id  INTEGER NOT NULL REFERENCES contestant(id) ON DELETE CASCADE,
  date           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  poster_id      INTEGER NOT NULL REFERENCES poster(id) ON DELETE RESTRICT,
  contest_id     INTEGER NOT NULL REFERENCES contest(id) ON DELETE RESTRICT,
  survey_item_id INTEGER NOT NULL REFERENCES survey_item(id) ON DELETE RESTRICT,
  value_selected INTEGER NOT NULL CHECK (value_selected BETWEEN 1 AND 5)
);
CREATE INDEX idx_answer_contestant_id ON answer(contestant_id);
CREATE INDEX idx_answer_poster_id ON answer(poster_id);
CREATE INDEX idx_answer_contest_item ON answer(contest_id, survey_item_id);
