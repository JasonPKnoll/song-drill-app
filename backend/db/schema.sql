-- User profiles. No authentication — the app is Tailscale-only (network
-- access is already gated), so this just partitions progress/stats between
-- people sharing the same install. Selected via a plain cookie (see
-- backend/handlers/profiles.go), not a password.
CREATE TABLE IF NOT EXISTS users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    display_name TEXT NOT NULL,
    color        TEXT NOT NULL DEFAULT '#a78bfa', -- accent color for the profile picker UI
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

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

-- SRS progress for vocab cards. Anki-style state machine (new -> learning ->
-- review, with relearning on a lapse from review) — see backend/srs/srs.go,
-- the single source of truth for the scheduling algorithm itself; this table
-- only stores whatever that package's State struct needs persisted.
CREATE TABLE IF NOT EXISTS vocab_progress (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id       INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    vocab_id      INTEGER NOT NULL REFERENCES vocab(id),
    state         TEXT NOT NULL DEFAULT 'new',            -- new | learning | review | relearning
    step_index    INTEGER NOT NULL DEFAULT 0,             -- position within the current learning/relearning steps
    ease_factor   REAL NOT NULL DEFAULT 2.5,              -- SM-2 ease, applied while in the review state
    interval_days REAL NOT NULL DEFAULT 0,                -- last computed review-state interval
    lapses        INTEGER NOT NULL DEFAULT 0,              -- times missed while in the review state
    seen          INTEGER NOT NULL DEFAULT 0,
    correct       INTEGER NOT NULL DEFAULT 0,
    due           TEXT NOT NULL DEFAULT (datetime('now')), -- full datetime: learning/relearning steps are seconds-scale
    last_seen     TEXT,
    introduced_at TEXT,                                   -- when this word was assigned into a daily new-word batch (nullable; see VocabDrillQueue's daily cap)
    UNIQUE(user_id, song_id, vocab_id)      -- progress is per profile, per song — not global
);

-- SRS progress for line cards (same state machine as vocab_progress).
CREATE TABLE IF NOT EXISTS line_progress (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    line_id       INTEGER NOT NULL REFERENCES lines(id) ON DELETE CASCADE,
    state         TEXT NOT NULL DEFAULT 'new',
    step_index    INTEGER NOT NULL DEFAULT 0,
    ease_factor   REAL NOT NULL DEFAULT 2.5,
    interval_days REAL NOT NULL DEFAULT 0,
    lapses        INTEGER NOT NULL DEFAULT 0,
    seen          INTEGER NOT NULL DEFAULT 0,
    correct       INTEGER NOT NULL DEFAULT 0,
    due           TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen     TEXT,
    introduced_at TEXT,                                   -- when this line was assigned into a daily new-line batch (nullable; see LineDrillQueue's daily cap)
    UNIQUE(user_id, line_id)
);
