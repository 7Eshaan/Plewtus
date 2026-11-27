# Plewtus Frontend

To build and run the full application with Docker Compose, follow the root [quickstart](../README.md#quickstart).

The Next.js App Router application provides an interactive metasearch demo and API/MCP quickstarts. Its `/api/search` route applies a process-local per-IP guard and forwards requests to the Go backend using a server-only API key.

## Run locally

Create `frontend/.env.local` from `.env.example`, set a backend URL and server-side API key, then start the app:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). Start the Go server from `backend/` as described in the root `README.md`; the frontend default backend URL is `http://localhost:8080` (override with `PLEWTUS_BACKEND_URL`).

## Commands

```bash
npm run lint
npm run build
npm run generate:types   # regenerate src/types/api.ts from backend/docs/openapi.yaml
npm run generate:theme   # regenerate src/app/m3-theme.css (Material 3 colors from the #8700ff seed)
npm run verify           # checks for the search proxy
```

The frontend IP guards are in-memory per Node process: `PLEWTUS_RATE_LIMIT_PER_MINUTE` (default 20) for `/api/search`, and a separate `PLEWTUS_FAVICON_RATE_LIMIT_PER_MINUTE` (default 300) for `/api/favicon`, so result icons never use up a visitor's searches. Production deployments that need a global quota should enforce it at a shared gateway/store. How the route identifies a visitor:

- On Vercel, it uses the platform's `x-vercel-forwarded-for` header.
- With `PLEWTUS_TRUST_PROXY_HEADERS=true` (only behind a reverse proxy you control), it uses `X-Real-IP` if that is a valid address, otherwise the last valid address in `X-Forwarded-For`.
- Otherwise forwarded headers are ignored and all visitors share one rate-limit bucket.
