#!/usr/bin/env bash
# Ingests every fixture in backend/testdata/ into a running song-drill API.
# Fixtures use invented lyrics only — see testdata/README.md.
set -euo pipefail

API="${SONG_DRILL_API:-http://localhost:30001}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$SCRIPT_DIR/../testdata"

for fixture in "$TESTDATA_DIR"/*.json; do
	echo "Ingesting $(basename "$fixture")..."
	curl -sf -X POST "$API/api/song-drill/songs/ingest" \
		-H "Content-Type: application/json" \
		-d @"$fixture" \
		| python3 -m json.tool
done

echo "Done. GET $API/api/song-drill/songs to see the library."
