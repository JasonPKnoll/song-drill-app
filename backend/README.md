# song-drill backend

Go API server for song-drill. Owns the SQLite database and all SRS logic.
Developer reference — see the root `README.md` for the project overview.

## Requirements

- Go 1.24+ (cgo required, for `mattn/go-sqlite3`)

## Running

```bash
go run main.go
```

Listens on `:30001` by default. The database file defaults to `song-drill.db`
in the current directory — created on first run, with schema migrations
applied automatically on every startup (see "Database migrations" below).

Override either with env vars:

```bash
SONG_DRILL_DB=/path/to/other.db SONG_DRILL_ADDR=:8080 go run main.go
```

## Building

```bash
go build -o song-drill-api .
./song-drill-api
```

## Project layout

```
main.go                     chi router setup, route registration, CORS
handlers/                   HTTP handlers (songs, drill, stats) — thin, delegate to db/
db/
  schema.sql                 DDL (source of truth: ../schema.md)
  db.go                       connection + migrations
  types.go                    Go structs for ingest payloads + API responses
  queries.go                  all SQL lives here
srs/srs.go                  SRS interval table (0,1,3,7,14,30,90 days) + streak update logic
scripts/
  seed.sh                    ingest backend/testdata/*.json into a running backend
  import_annotator_output.py  import lyrics-annotator's output/ directory
testdata/                   invented demo song fixtures — no real lyrics, ever
```

## Database migrations

`db.Open` applies `schema.sql`, then runs `migrate()`. This split matters:
`CREATE TABLE IF NOT EXISTS` in `schema.sql` only affects brand-new databases
— it does **not** add new columns to a table that already exists. `migrate()`
checks for each column added after the fact via `PRAGMA table_info` and adds
it with `ALTER TABLE` if missing, so existing databases (and their data) pick
up schema changes without needing to be recreated.

**When you add a column to `schema.sql`, also add an entry to the
`columnMigrations` slice in `db.go`** — otherwise anyone with an
already-existing database silently won't get it, and will hit a `no such
column` error on first use.

## Seeding / importing data

```bash
# invented demo fixtures, for local development and UI testing
./scripts/seed.sh

# real songs from lyrics-annotator's output/ directory (default: ../lyrics-annotator/output)
python3 scripts/import_annotator_output.py           # skips songs already imported
python3 scripts/import_annotator_output.py --replace # delete + re-ingest if the song already exists
python3 scripts/import_annotator_output.py --dry-run # preview only, no writes
```

Both scripts talk to a running backend over HTTP — start `go run main.go`
first.

## API routes

All under `/api/song-drill`:

| Method | Path                | Purpose                                     |
|--------|---------------------|----------------------------------------------|
| POST   | `/songs/ingest`     | Ingest a song JSON payload                    |
| GET    | `/songs`            | List songs with progress stats                |
| GET    | `/songs/{id}`       | Song detail (lines + vocab)                   |
| DELETE | `/songs/{id}`       | Delete a song (cascades to lines/vocab/progress) |
| GET    | `/songs/{id}/lines` | Ordered lines for a song                      |
| GET    | `/drill/vocab`      | SRS vocab queue (`?song_id=&limit=`)          |
| GET    | `/drill/lines`      | SRS line queue (`?song_id=&limit=`)           |
| POST   | `/drill/result`     | Record a drill answer, update SRS             |
| GET    | `/stats`            | Overall progress stats                        |

## Manual testing

```bash
curl -X POST http://localhost:30001/api/song-drill/songs/ingest \
  -H "Content-Type: application/json" \
  -d @testdata/demo_song.json

curl http://localhost:30001/api/song-drill/songs
```
