-- Songs
CREATE TABLE IF NOT EXISTS songs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    artist      TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT 'ja',
    notes       TEXT,
    created_at  TEXT NOT NULL DEFAULT (date('now'))
);

-- Ordered lines (lossless, choruses repeat)
CREATE TABLE IF NOT EXISTS lines (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id     INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,             -- 0-indexed, preserves order
    text        TEXT NOT NULL,               -- plain Japanese
    reading     TEXT NOT NULL,               -- full kana reading
    furi        TEXT NOT NULL,               -- 漢字[よみ] markup
    literal     TEXT NOT NULL,               -- word-for-word English
    natural     TEXT NOT NULL,               -- fluent English translation
    contextual  TEXT NOT NULL,               -- emotionally framed translation
    grammar_note TEXT,
    section     TEXT,                        -- e.g. "Verse 1", "Chorus" (nullable)
    UNIQUE(song_id, position)
);

-- Global canonical vocab (one row per unique word across all songs)
CREATE TABLE IF NOT EXISTS vocab (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    surface     TEXT NOT NULL,               -- dictionary/display form
    reading     TEXT NOT NULL,               -- hiragana reading
    furi        TEXT NOT NULL,               -- 漢字[よみ] markup
    pos         TEXT NOT NULL,               -- part of speech
    base_meaning TEXT NOT NULL,              -- dictionary meaning
    UNIQUE(surface, reading)                 -- dedup constraint
);

-- Per-song vocab join (carries song-specific context)
CREATE TABLE IF NOT EXISTS song_vocab (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id             INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    vocab_id            INTEGER NOT NULL REFERENCES vocab(id),
    context_meaning     TEXT NOT NULL,       -- meaning in this song's context
    first_line_position INTEGER NOT NULL,    -- position of first occurrence
    UNIQUE(song_id, vocab_id)
);

-- Per-line word occurrences (for line-level word highlighting)
CREATE TABLE IF NOT EXISTS line_words (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    line_id     INTEGER NOT NULL REFERENCES lines(id) ON DELETE CASCADE,
    vocab_id    INTEGER NOT NULL REFERENCES vocab(id),
    position    INTEGER NOT NULL             -- word order within the line
);

-- SRS progress for vocab cards
CREATE TABLE IF NOT EXISTS vocab_progress (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id     INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    vocab_id    INTEGER NOT NULL REFERENCES vocab(id),
    streak      INTEGER NOT NULL DEFAULT 0,
    seen        INTEGER NOT NULL DEFAULT 0,
    correct     INTEGER NOT NULL DEFAULT 0,
    next_review TEXT NOT NULL DEFAULT (date('now')),
    last_seen   TEXT,
    UNIQUE(song_id, vocab_id)               -- progress is per song, not global
);

-- SRS progress for line cards
CREATE TABLE IF NOT EXISTS line_progress (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    line_id     INTEGER NOT NULL REFERENCES lines(id) ON DELETE CASCADE,
    streak      INTEGER NOT NULL DEFAULT 0,
    seen        INTEGER NOT NULL DEFAULT 0,
    correct     INTEGER NOT NULL DEFAULT 0,
    next_review TEXT NOT NULL DEFAULT (date('now')),
    last_seen   TEXT,
    UNIQUE(line_id)
);
