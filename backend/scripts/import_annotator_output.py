#!/usr/bin/env python3
"""Imports lyrics-annotator output JSON files into a running song-drill API.

Skips checkpoint files (anything with "checkpoint" in the name, and dotfiles)
and skips songs already present in the database, matched by (title, artist).
Pass --replace to delete and re-ingest a matching existing song instead of
skipping it — useful if you imported while lyrics-annotator was still running
and want the now-complete version.
"""
import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

# backend/scripts/<this file> -> backend -> song-learning-app -> Iori -> lyrics-annotator/output
DEFAULT_OUTPUT_DIR = Path(__file__).resolve().parent.parent.parent.parent / "lyrics-annotator" / "output"
DEFAULT_API = "http://localhost:30001"


def fetch_existing_songs(api_base: str) -> list[dict]:
	with urllib.request.urlopen(f"{api_base}/api/song-drill/songs") as resp:
		return json.load(resp) or []


def ingest_song(api_base: str, payload: dict) -> int:
	req = urllib.request.Request(
		f"{api_base}/api/song-drill/songs/ingest",
		data=json.dumps(payload).encode("utf-8"),
		headers={"Content-Type": "application/json"},
		method="POST",
	)
	with urllib.request.urlopen(req) as resp:
		return json.load(resp)["song_id"]


def delete_song(api_base: str, song_id: int) -> None:
	req = urllib.request.Request(f"{api_base}/api/song-drill/songs/{song_id}", method="DELETE")
	with urllib.request.urlopen(req):
		pass


def find_candidate_files(output_dir: Path) -> list[Path]:
	candidates = []
	for path in sorted(output_dir.glob("*.json")):
		name = path.name
		if name.startswith(".") or "checkpoint" in name.lower():
			continue
		candidates.append(path)
	return candidates


def main() -> None:
	parser = argparse.ArgumentParser(description=__doc__)
	parser.add_argument(
		"--output-dir",
		type=Path,
		default=DEFAULT_OUTPUT_DIR,
		help=f"lyrics-annotator output directory (default: {DEFAULT_OUTPUT_DIR})",
	)
	parser.add_argument("--api", default=DEFAULT_API, help=f"song-drill API base URL (default: {DEFAULT_API})")
	parser.add_argument("--dry-run", action="store_true", help="show what would be imported without ingesting")
	parser.add_argument(
		"--replace",
		action="store_true",
		help="delete and re-ingest a matching existing song instead of skipping it "
		"(e.g. you imported before lyrics-annotator finished, and now have a complete version)",
	)
	args = parser.parse_args()

	if not args.output_dir.is_dir():
		print(f"Output directory not found: {args.output_dir}", file=sys.stderr)
		print("Pass --output-dir to point at lyrics-annotator's output/ folder.", file=sys.stderr)
		sys.exit(1)

	try:
		existing = {(s["title"], s["artist"]): s["id"] for s in fetch_existing_songs(args.api)}
	except urllib.error.URLError as e:
		print(f"Could not reach song-drill API at {args.api}: {e}", file=sys.stderr)
		print("Is the backend running?", file=sys.stderr)
		sys.exit(1)

	candidates = find_candidate_files(args.output_dir)
	if not candidates:
		print(f"No importable .json files found in {args.output_dir}")
		return

	imported = skipped = replaced = failed = 0
	for path in candidates:
		try:
			payload = json.loads(path.read_text(encoding="utf-8"))
		except json.JSONDecodeError as e:
			print(f"SKIP (invalid JSON): {path.name} — {e}")
			failed += 1
			continue

		song = payload.get("song", {})
		title, artist = song.get("title"), song.get("artist")
		if not title or not artist:
			print(f"SKIP (missing song.title/artist): {path.name}")
			failed += 1
			continue

		key = (title, artist)
		is_replace = key in existing and args.replace

		if key in existing and not args.replace:
			print(f"SKIP (already imported): {title} — {artist} ({path.name})")
			skipped += 1
			continue

		if args.dry_run:
			verb = "WOULD REPLACE" if is_replace else "WOULD IMPORT"
			print(f"{verb}: {title} — {artist} ({path.name})")
			if is_replace:
				replaced += 1
			else:
				imported += 1
			existing[key] = existing.get(key, -1)
			continue

		if is_replace:
			try:
				delete_song(args.api, existing[key])
			except urllib.error.URLError as e:
				print(f"FAILED (could not delete existing song for replace): {title} — {artist}: {e}")
				failed += 1
				continue

		try:
			song_id = ingest_song(args.api, payload)
		except urllib.error.HTTPError as e:
			print(f"FAILED: {title} — {artist} ({path.name}): {e.code} {e.read().decode('utf-8', 'replace')}")
			failed += 1
			continue
		except urllib.error.URLError as e:
			print(f"FAILED: {title} — {artist} ({path.name}): {e}")
			failed += 1
			continue

		if is_replace:
			print(f"REPLACED: {title} — {artist} ({path.name}) -> song_id={song_id}")
			replaced += 1
		else:
			print(f"IMPORTED: {title} — {artist} ({path.name}) -> song_id={song_id}")
			imported += 1
		existing[key] = song_id

	print(f"\nDone. Imported: {imported}, Replaced: {replaced}, Skipped (already existed): {skipped}, Failed: {failed}")


if __name__ == "__main__":
	main()
