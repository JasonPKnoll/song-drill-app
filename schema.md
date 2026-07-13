# song-drill Schema

Source of truth for the JSON input shape and SQLite database schema.
Both lyrics-annotator (producer) and song-drill (consumer) reference this document.

---

## JSON Input Shape

This is what `lyrics-annotator` produces and what song-drill ingests.
One file per song.

```json
{
  "song": {
    "title": "string",
    "artist": "string",
    "language": "ja",
    "notes": "optional string — emotional frame, context notes"
  },
  "lines": [
    {
      "position": 0,
      "text": "夜の街をひとりで歩く",
      "reading": "よるのまちをひとりであるく",
      "furi": "夜[よる]の街[まち]をひとりで歩[ある]く",
      "literal": "The night city alone walk",
      "natural": "Walking alone through the night city",
      "contextual": "Wandering alone through streets that feel emptier than they should",
      "grammar_note": "をひとりで — を marks the path of movement, ひとりで = alone/by oneself",
      "section": "optional string — e.g. \"Verse 1\", \"Chorus\", \"Outro\"",
      "words": [
        {
          "surface": "夜",
          "reading": "よる",
          "furi": "夜[よる]",
          "pos": "noun",
          "base_meaning": "night",
          "context_meaning": "the kind of night you walk through alone"
        },
        {
          "surface": "街",
          "reading": "まち",
          "furi": "街[まち]",
          "pos": "noun",
          "base_meaning": "town / city street",
          "context_meaning": "the city as backdrop to isolation"
        },
        {
          "surface": "歩く",
          "reading": "あるく",
          "furi": "歩[ある]く",
          "pos": "verb",
          "base_meaning": "to walk",
          "context_meaning": "to drift through, not walk with purpose"
        }
      ]
    }
  ],
  "vocab": [
    {
      "surface": "夜",
      "reading": "よる",
      "furi": "夜[よる]",
      "pos": "noun",
      "base_meaning": "night",
      "context_meaning": "the kind of night you walk through alone",
      "first_line_position": 0
    }
  ]
}
```

### Key points about the JSON shape

- `lines` is **ordered and lossless** — chorus repeats appear multiple times.
  Do not deduplicate. Position 0-indexed, sequential.
- `vocab` is **deduplicated at song level** — one entry per unique `(surface, reading)`.
  `first_line_position` points to where the word first appears.
- `line.words` has full word data per occurrence (for line-level context).
- `vocab` array has deduplicated words with song-level `context_meaning`.
- `furi` strings use `漢字[よみ]` markup throughout.
- All fields except `notes`, `grammar_note`, and `section` are required.
- Some lines may be **content-less** (empty `reading`/`furi`, null `literal`/`natural`/
  `contextual`, empty `words`) — this happens when the upstream scrape picks up
  non-lyric page content (e.g. "You might also like", site credits). These are
  still ingested losslessly (see Notes on design decisions below), not filtered
  or rejected.

---

## SQLite Schema (DDL)

Use this exactly. Do not modify column names or constraints.

```sql
-- User profiles. No authentication — the app is Tailscale-only (network
-- access is already gated), so this just partitions progress/stats between
-- people sharing the same install. Selected via a plain cookie, not a
-- password — see the Profiles section below.
CREATE TABLE users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    display_name TEXT NOT NULL,
    color        TEXT NOT NULL DEFAULT '#a78bfa',
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Songs
CREATE TABLE songs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    artist      TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT 'ja',
    notes       TEXT,
    created_at  TEXT NOT NULL DEFAULT (date('now'))
);

-- Ordered lines (lossless, choruses repeat)
CREATE TABLE lines (
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
CREATE TABLE vocab (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    surface     TEXT NOT NULL,               -- dictionary/display form
    reading     TEXT NOT NULL,               -- hiragana reading
    furi        TEXT NOT NULL,               -- 漢字[よみ] markup
    pos         TEXT NOT NULL,               -- part of speech
    base_meaning TEXT NOT NULL,              -- dictionary meaning
    UNIQUE(surface, reading)                 -- dedup constraint
);

-- Per-song vocab join (carries song-specific context)
CREATE TABLE song_vocab (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    song_id             INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    vocab_id            INTEGER NOT NULL REFERENCES vocab(id),
    context_meaning     TEXT NOT NULL,       -- meaning in this song's context
    first_line_position INTEGER NOT NULL,    -- position of first occurrence
    UNIQUE(song_id, vocab_id)
);

-- Per-line word occurrences (for line-level word highlighting)
CREATE TABLE line_words (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    line_id     INTEGER NOT NULL REFERENCES lines(id) ON DELETE CASCADE,
    vocab_id    INTEGER NOT NULL REFERENCES vocab(id),
    position    INTEGER NOT NULL             -- word order within the line
);

-- SRS progress for vocab cards. Anki-style state machine (new -> learning ->
-- review, with relearning on a lapse from review) — see backend/srs/srs.go.
CREATE TABLE vocab_progress (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id       INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    vocab_id      INTEGER NOT NULL REFERENCES vocab(id),
    state         TEXT NOT NULL DEFAULT 'new',            -- new | learning | review | relearning
    step_index    INTEGER NOT NULL DEFAULT 0,             -- position within the current learning/relearning steps
    ease_factor   REAL NOT NULL DEFAULT 2.5,              -- SM-2 ease, applied while in the review state
    interval_days REAL NOT NULL DEFAULT 0,                -- last computed review-state interval
    lapses        INTEGER NOT NULL DEFAULT 0,             -- times missed while in the review state
    seen          INTEGER NOT NULL DEFAULT 0,
    correct       INTEGER NOT NULL DEFAULT 0,
    due           TEXT NOT NULL DEFAULT (datetime('now')), -- full datetime: learning/relearning steps are minutes-scale
    last_seen     TEXT,
    UNIQUE(user_id, song_id, vocab_id)      -- progress is per profile, per song — not global
);

-- SRS progress for line cards (same state machine as vocab_progress).
CREATE TABLE line_progress (
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
    UNIQUE(user_id, line_id)
);
```

---

## SRS Algorithm

Anki-style scheduler (classic SM-2-derived, not FSRS), reduced to a plain
correct/incorrect grade — no Hard/Easy buttons, per Anki's own community
guidance that they're hard to grade consistently and mostly add noise.
Implemented in Go (`backend/srs/srs.go`) and nowhere else — the frontend
never computes SRS state, it only calls API endpoints and renders results.

**Stages:** `new` → `learning` → `review`, with a miss in `review` dropping
the card into `relearning` before it re-graduates back to `review`.

**Learning / relearning** (same-day, minutes-scale):
- Learning steps: `1m, 10m` — a correct answer advances one step; passing
  the last step graduates the card into `review` with a 1-day interval.
- Relearning step: `10m` — same mechanic, entered on a review-stage lapse;
  graduating back out restores the interval assigned at the moment of lapse
  (see below), not the standard 1-day graduating interval.
- A miss during learning/relearning resets to the **first** step — same-day,
  shown again soon — not out of the phase entirely. This is the "resets
  progress for that word for the day" behavior.

**Review** (day-scale, ease-factor driven):
- Starting ease: 2.5 (250%). Passing a review multiplies the interval by the
  current ease (`interval *= ease`); ease itself doesn't change on a pass
  (no Easy button to raise it further).
- A miss (lapse): ease drops by 20 percentage points (floor 130%), the
  interval that took however long to earn is forfeited and reset to the
  1-day minimum, and the card drops into `relearning`.
- `lapses` is tracked per card (for a future leech indicator, e.g. flagging
  after 8 lapses, matching Anki's default) but nothing currently acts on it.

**Mastered** (a display/stats concept, not part of the algorithm): a card
in the `review` stage with `interval_days >= 30`.

---

## Profiles

No authentication. The app is Tailscale-only, so network access is already
gated — profiles just let more than one person share an install and keep
separate SRS progress/stats, picked via a plain, unsigned `song_drill_user`
cookie (see `backend/handlers/profiles.go`), not a password.

- Songs/vocab/lines are global, shared content — only `vocab_progress` and
  `line_progress` are scoped per profile.
- Middleware resolves the active profile from the cookie on every request,
  falling back to the earliest-created profile if the cookie is missing or
  names a profile that's since been deleted (e.g. from another tab).
- There must always be at least one profile — deleting the last remaining
  one is rejected at the query layer.

---

## Ingest logic (pseudocode)

When `POST /api/song-drill/songs/ingest` receives a JSON file:

```
1. Insert song → get song_id
2. For each line in lines[]:
   - Insert into lines (song_id, position, text, reading, furi, ...)
   - get line_id
   - For each word in line.words[]:
     - INSERT OR IGNORE into vocab (surface, reading, furi, pos, base_meaning)
     - get vocab_id
     - Insert into line_words (line_id, vocab_id, position)
3. For each entry in vocab[]:
   - INSERT OR IGNORE into vocab (surface, reading, furi, pos, base_meaning)
   - get vocab_id
   - INSERT OR IGNORE into song_vocab (song_id, vocab_id, context_meaning, first_line_position)
```

Use `INSERT OR IGNORE` for vocab to respect the `UNIQUE(surface, reading)` constraint.
Use `INSERT OR REPLACE` or `ON CONFLICT UPDATE` for song_vocab if re-ingesting.

---

## Notes on design decisions

- **Vocab progress is per-profile, per-song** (`UNIQUE(user_id, song_id, vocab_id)`)
  not global. The same word in two songs gets two separate SRS tracks, because
  the context meanings are different — and the same word in the same song gets
  a separate track per profile.
- **Line progress is per-profile** (`UNIQUE(user_id, line_id)`) — a line already
  belongs to one song, so no song_id is needed, but each profile still tracks
  it independently.
- **Chorus repeats are preserved in `lines`** because the song reader needs to
  display the song faithfully. SRS line drill deduplicates naturally since
  repeated lines have different `line_id` values but the same `text` — this
  is acceptable, the user may see similar cards.
- **`vocab.base_meaning`** is what's shown on the vocab drill card reveal — a
  plain dictionary definition. `song_vocab.context_meaning` (the per-song
  emotional framing) currently only backs vocab search matching in the song
  detail page, not its own visible field.
- **Content-less lines are ingested, not filtered** — `lines` stays fully lossless
  regardless of upstream scrape quality. They're excluded from the *line drill
  SRS queue* (`WHERE reading != ''`), since there's nothing to quiz, but they
  still appear in the song reader — grouped by `section` — rendered as plain
  text with no furigana/reveal interaction, since there's no reading to reveal.
