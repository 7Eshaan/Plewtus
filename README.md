# Plewtus

Plewtus is a metasearch API with a Next.js playground. It queries DuckDuckGo, Bing, and Yahoo, combines results with Reciprocal Rank Fusion, and can extract page content as Markdown, text, or highlights.

## Run the full stack with Docker

Requirements: Docker Desktop with Docker Compose.

```bash
cp .env.example .env
```

Edit `.env` and replace `PLEWTUS_API_KEY` with a long random value. Then build and start both services:

```bash
docker compose up --build -d
```

Open [http://localhost:3000](http://localhost:3000) for the web app. The API is available at `http://localhost:8080`.

To check the API directly, load the key from `.env` into your shell and send a search request:

```bash
set -a
. ./.env
set +a

curl -sS http://localhost:8080/search \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $PLEWTUS_API_KEY" \
  -d '{"query":"golang concurrency patterns","limit":3}'
```

To request Markdown extraction, include an `extract` object:

```bash
curl -sS http://localhost:8080/search \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $PLEWTUS_API_KEY" \
  -d '{"query":"golang concurrency patterns","limit":3,"extract":{"enabled":true,"mode":"markdown","max_pages":2,"max_chars_per_page":4000}}'
```

The extracted content appears under each result's `content.markdown`. The API key is shared between the services through Compose and is required by the backend.

Useful commands:

```bash
docker compose logs -f       # follow service logs
docker compose ps             # show service and health status
docker compose down           # stop and remove the containers
docker compose up --build -d  # rebuild after changing code
```

You can change the published ports with `FRONTEND_PORT` and `BACKEND_PORT` in `.env`. Do not commit `.env` or use the example key for a public deployment. The API's cache, quotas, and circuit-breaker state are in memory and reset when the backend restarts.

## Run services without Docker

See [backend/README.md](backend/README.md) and [frontend/README.md](frontend/README.md) for local development instructions.
