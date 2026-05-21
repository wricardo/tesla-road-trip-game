# ADR 0001: Frontend Stack for Spectator UI v2

**Status:** Accepted  
**Date:** 2026-05-19  
**Deciders:** wricardo

---

## Context

Replacing old embedded HTML/JS game UI with a public spectator app hosted on AWS (S3 + CloudFront). Requirements:

- Read-only spectator view — no player controls
- Real-time updates via WebSocket (GraphQL subscriptions from gqlgen backend)
- Static hosting (S3/CloudFront, no SSR runtime)
- Multiple public users watching concurrently
- AI-learning focus: clean, legible, content-heavy pages
- Not React (per product decision)

---

## Candidates Evaluated

| | SvelteKit | SolidJS | htmx + Alpine | plain Vite + Svelte |
|---|---|---|---|---|
| Bundle size | ~50–80 KB | ~20–30 KB | ~15 KB | ~30–50 KB |
| Static export | ✅ adapter-static | ✅ static build | ✅ | ✅ |
| GraphQL + WS subs | graphql-ws + urql | same | polling/manual | same |
| Routing | built-in | tanstack router | attribute-based | none (add manually) |
| Component model | reactive, idiomatic | fine-grained signals | server-oriented | same as SvelteKit |
| Go team familiarity | low, but simple | lower | higher | same as SvelteKit |
| Ecosystem maturity | stable (1.x) | stable (1.x) | stable | same |

---

## Decision: SvelteKit with `adapter-static`

**Why:**

1. **Static export**: `@sveltejs/adapter-static` produces pure HTML/JS/CSS into `frontend/build/` — drop directly to S3.
2. **Svelte's reactivity model is minimal and readable** — no virtual DOM overhead, no hooks rules. Easy to maintain without deep framework knowledge.
3. **File-based routing built-in** — pages at `src/routes/+page.svelte`, `src/routes/watch/[id]/+page.svelte`. No extra router library.
4. **graphql-ws** integrates as a plain JS import. `urql` (or `@urql/svelte`) gives typed queries with Svelte store bindings out of the box.
5. **TypeScript + Tailwind** both supported with minimal config.
6. **SolidJS** is lighter but ecosystem thinner; htmx model fights against real-time subscription patterns.

---

## Stack

| Layer | Choice |
|---|---|
| Framework | SvelteKit 2.x |
| Adapter | `@sveltejs/adapter-static` |
| Styling | Tailwind CSS v4 |
| GraphQL client | `@urql/svelte` + `graphql-ws` |
| Code generation | `graphql-codegen` (typed operations) |
| TypeScript | Yes |
| Build output | `frontend/build/` |
| Node for build | 22 (via mise) |

---

## Environment Variables

```
VITE_GRAPHQL_URL=https://api.example.com/graphql
VITE_WS_URL=wss://api.example.com/graphql
```

Both injected at build time. Local dev defaults to `http://localhost:8080/graphql` and `ws://localhost:8080/graphql`.

---

## Consequences

- `frontend/` lives inside this repo alongside Go server.
- New pages are Svelte components; no Go template changes needed.
- Old `static/` dir (game.js, index.html) will be removed in cleanup ticket `.15` once new UI ships.
- gqlgen needs GraphQL subscriptions added (ticket `.2`) before WS features work in dev.
