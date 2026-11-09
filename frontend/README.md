# Plewtus Frontend

To build and run the full application with Docker Compose, follow the root [Docker guide](../README.md#run-the-full-stack-with-docker).

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
npm run generate:types
```

The frontend IP guard is in-memory per Node process. Production deployments that need a global quota should enforce it at a shared gateway/store. The deployment proxy must append or overwrite `X-Forwarded-For`; the route uses the last valid address in that chain.
