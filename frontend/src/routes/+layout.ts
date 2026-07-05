// Full SPA mode: song IDs only exist after runtime ingest, so nothing can be
// prerendered at build time. adapter-static serves everything through the
// 200.html fallback and the app renders entirely client-side.
export const ssr = false;
export const prerender = false;
