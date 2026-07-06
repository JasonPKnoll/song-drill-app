# song-drill frontend

SvelteKit + TypeScript + Tailwind. Builds to a fully static SPA
(`@sveltejs/adapter-static`, `ssr = false`) — the backend is a separate Go
process, called over `/api/song-drill/...` (see `src/lib/api.ts`). Developer
reference — see the root `README.md` for the project overview.

## Requirements

- Node 20+
- The backend running on `:30001` (see `../backend/README.md`) — the dev
  server proxies `/api` to it (`vite.config.ts` → `server.proxy`).

## Running (development)

```bash
npm install
npm run dev
```

Opens on `:5173`.

### `npm run dev` feels slow — what's going on, and how to make it fast

`vite dev` normally compiles each route's JS the first time you visit it in a
session, which shows up as ~0.5–1s of lag on the first click into any given
page. `vite.config.ts` sets `server.warmup` to pre-compile every route at
startup instead, so this shouldn't happen after a fresh `npm run dev` — if it
still feels slow, restart the dev server (warmup runs once at boot, not per
request).

**If you're not editing code and just want to use the app quickly**, skip dev
mode's HMR/on-demand-compile overhead entirely and run the production build
instead:

```bash
npm run build
npm run preview
```

Serves on `:4173`, already bundled and minified — no on-demand compilation.
Navigation is close to instant (measured ~50–75ms per page vs. ~650ms on a
cold `vite dev` route visit). The tradeoff: it's a static snapshot of
whatever you last built — re-run `npm run build` to pick up code changes,
there's no live reload.

## Type-checking

```bash
npm run check
```

## Building for deployment

```bash
npm run build
```

Output goes to `build/` — a static site with `200.html` as the SPA fallback
(every route is client-rendered; see `svelte.config.js`). Deploy behind a
server that proxies `/api/song-drill/` to the Go backend (see the root
`README.md`'s Nginx example) — the frontend only ever calls relative paths,
never `localhost:30001` directly, so this proxy is required wherever it's
served from.

## Project layout

```
src/
  routes/
    +layout.svelte, +layout.ts   root shell; ssr=false / prerender=false (required for adapter-static + dynamic song IDs)
    +page.svelte, +page.ts        song library home
    songs/[id]/                   song detail, reader, browse-vocab
    drill/vocab/, drill/lines/    SRS drill screens
  lib/
    api.ts                        typed fetch wrappers for every backend endpoint
    furigana.ts                   漢字[よみ] markup parser
    components/                   Furigana, DrillCard, SongCard, BackLink, VocabFlipCard
```

## Why data fetching happens in `load()`, not `onMount`

Every page fetches its data in a `+page.ts` `load()` function, not the
component's `onMount`. This is what makes `data-sveltekit-preload-data="hover"`
(set in `app.html`) actually do anything — SvelteKit only preloads data ahead
of a click for routes using `load()`. Fetching in `onMount` instead means
every navigation shows a blank loading flash with no way to prefetch it. If
you add a new route, fetch its data in a `+page.ts` and thread the `fetch` it
gives you into `api.ts` calls (see any existing `+page.ts` for the pattern) —
don't fetch inline in the `.svelte` file.
