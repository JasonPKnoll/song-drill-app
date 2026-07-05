# Test fixtures

Invented demo songs for local development and testing. No real song lyrics —
per project rule, only fabricated content lives here.

- `demo_song.json` — 紙飛行機 ("Paper Airplane"), 4 lines, 6 vocab words, includes a repeated chorus line.
- `demo_song_2.json` — 雨音 ("Sound of Rain"), 4 lines, 8 vocab words. Reuses the word 風 (wind) from `demo_song.json` with a different context meaning, to exercise the per-song vocab dedup behavior described in `song_drill_schema.md`.

Ingest both at once with `backend/scripts/seed.sh` (backend must already be running).
