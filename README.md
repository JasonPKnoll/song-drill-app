# song-drill

Japanese language learning app that teaches vocabulary and sentence patterns
through songs. Part of the [Iori](https://github.com/jasonknoll/dashboard)
personal dashboard ecosystem.

**Status:** In development

---

## What it does

song-drill consumes structured JSON produced by
[lyrics-annotator](https://github.com/jasonknoll/lyrics-annotator), stores it
in SQLite, and lets you study vocabulary and lines from those songs using spaced
repetition.

Words are learned with the **contextual meaning they carry in that specific
song** — not dictionary definitions. A word that means "to fall" in a dictionary
means "falling out of love" in a breakup song. That's the meaning stored and
drilled.

---

## Study modes

**Vocab drill** — SRS flashcard per word. Front shows the Japanese word. Reveal
shows furigana, the song-specific context meaning, and an example line.

**Line drill** — SRS flashcard per line. Front shows the full Japanese sentence.
Reveal shows furigana, natural English, and a grammar note if present.

**Song reader** — Read through all lines of a song in order. No SRS, just
reading practice. Tap to reveal furigana and translations per line.

---

## Stack

| Layer | Technology |
|---|---|
| Frontend | SvelteKit + TypeScript + Tailwind (`@sveltejs/adapter-static`) |
| Backend | Go + chi router |
| Database | SQLite (`mattn/go-sqlite3`) |
| Served by | Nginx on Raspberry Pi 4 |
| Network | Tailscale (private, no public exposure) |

---

## Input format

song-drill ingests JSON produced by `lyrics-annotator`. The exact shape is
documented in `song_drill_schema.md`. Do not manually create ingest files — run
`lyrics-annotator` to produce them.

---

## Running locally

### Backend
```bash
cd backend
go mod download
go run main.go
# API listens on :30001
```

### Frontend
```bash
cd frontend
npm install
npm run dev
# Dev server on :5173
```

### Ingest a song
```bash
curl -X POST http://localhost:30001/api/song-drill/songs/ingest \
  -H "Content-Type: application/json" \
  -d @path/to/song_output.json
```

---

## Deployment

Built as a static SvelteKit export served by Nginx on the Pi. The Go binary runs
alongside as a systemd service. Accessible via Tailscale at the Pi's hostname.

The frontend calls the API with relative paths (`/api/song-drill/...`), so Nginx
must proxy that path to the Go backend — the browser has no way to reach the
backend's `localhost:30001` directly once it's loaded the page from the Pi's
Tailscale address. Add to the Nginx server block:

```nginx
location /api/song-drill/ {
    proxy_pass http://127.0.0.1:30001;
}
```

```bash
# frontend
cd frontend && npm run build
rsync -avz build/ pi@raspberrypi.local:/var/www/song-drill/

# backend
cd backend && go build -o song-drill-api .
scp song-drill-api pi@raspberrypi.local:/usr/local/bin/
```

---

## Ecosystem

```
lyrics-annotator   → produces song JSON (upstream, separate repo)
song-drill         → this repo, consumes JSON, serves the study app
dashboard          → shell landing page that links to this app
```

song-drill owns the database schema. lyrics-annotator owns the JSON output shape.
Those two contracts meet at `song_drill_schema.md`.