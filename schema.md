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

-- SRS progress for vocab cards
CREATE TABLE vocab_progress (
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
CREATE TABLE line_progress (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    line_id     INTEGER NOT NULL REFERENCES lines(id) ON DELETE CASCADE,
    streak      INTEGER NOT NULL DEFAULT 0,
    seen        INTEGER NOT NULL DEFAULT 0,
    correct     INTEGER NOT NULL DEFAULT 0,
    next_review TEXT NOT NULL DEFAULT (date('now')),
    last_seen   TEXT,
    UNIQUE(line_id)
);
```

---

## SRS Intervals

```
Streak 0 → today (new or just missed)
Streak 1 → +1 day
Streak 2 → +3 days
Streak 3 → +7 days
Streak 4 → +14 days
Streak 5 → +30 days  ← mastered threshold
Streak 6 → +90 days
```

Implemented in Go (`backend/srs/srs.go`). Not in the frontend.

```go
var intervals = []int{0, 1, 3, 7, 14, 30, 90}

func NextReview(streak int) string {
    days := intervals[min(streak, len(intervals)-1)]
    return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}
```

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

- **Vocab progress is per-song** (`UNIQUE(song_id, vocab_id)`) not global.
  The same word in two songs gets two separate SRS tracks, because the context
  meanings are different.
- **Line progress is global** (`UNIQUE(line_id)`) because a line belongs to one
  song — there's no ambiguity.
- **Chorus repeats are preserved in `lines`** because the song reader needs to
  display the song faithfully. SRS line drill deduplicates naturally since
  repeated lines have different `line_id` values but the same `text` — this
  is acceptable, the user may see similar cards.
- **`song_vocab.context_meaning`** is the primary learning hook — this is what
  gets shown on the vocab drill card reveal, not `vocab.base_meaning`.
- **Content-less lines are ingested, not filtered** — `lines` stays fully lossless
  regardless of upstream scrape quality. They're excluded from the *line drill
  SRS queue* (`WHERE reading != ''`), since there's nothing to quiz, but they
  still appear in the song reader — grouped by `section` — rendered as plain
  text with no furigana/reveal interaction, since there's no reading to reveal.
