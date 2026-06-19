# EasyBook

Multi-tenant SaaS platform that automates booking management for padel courts via WhatsApp, using an AI agent orchestrated with n8n.

## Stack

| Component | Technology |
|---|---|
| Control Plane API | Go 1.22 + Chi + pgx/v5 |
| Hot-reload (dev) | Air |
| Dashboard | React 18 + Vite + TypeScript |
| Messaging Orchestration | n8n (self-hosted) |
| Database | PostgreSQL 16 (schema-per-tenant + RLS) |
| AI Agent | GPT-4o-mini · Ollama · Mock (switchable via `LLM_PROVIDER`) |
| Infrastructure | Hetzner VPS + Docker Compose |

## Repository Structure

```
easybook/
├── apps/
│   ├── control-plane/   # Go — main API, provisioning, AI agent
│   │   ├── Dockerfile.dev
│   │   └── .air.toml
│   ├── dashboard/        # React — club management UI
│   │   └── Dockerfile.dev
│   └── n8n-workflows/    # n8n workflow JSON exports
├── packages/
│   └── database/         # Prisma schema + migrations (SQL schema source of truth)
├── infra/
│   ├── docker/
│   │   ├── docker-compose.dev.yml    # Postgres + n8n only (bare-host dev)
│   │   └── docker-compose.test.yml   # Full stack in Docker (recommended)
│   ├── hetzner/           # cloud-init, VPS setup
│   └── scripts/           # provisioning utilities
├── docs/
│   ├── ADR/               # architecture decision records
│   └── runbooks/          # operational guides
├── .env.example           # template — copy to .env.test for local dev
└── .github/workflows/     # CI/CD
```

> **Architecture note:** the database schema lives in `packages/database` using Prisma solely as a versioned migration tool (not as a runtime ORM). The Go `control-plane` uses `sqlc` to generate type-safe code from plain SQL against the same schema.

---

## Quickstart

### Option 1 — Docker (recommended, nothing to install locally)

```bash
# 0. Clone the repo
git clone <repo-url> easybook && cd easybook

# 1. Copy the env template (already committed as .env.test — skip if it exists)
cp .env.example .env.test

# 2. Start the full stack (builds app images on first run — takes ~1 min)
docker compose -f infra/docker/docker-compose.test.yml up
```

> **Note:** `control-plane` and `dashboard` images are built locally from `Dockerfile.dev` — no pre-built images to pull. Subsequent runs use the Docker layer cache and start in seconds.

That's it. The stack starts with `LLM_PROVIDER=mock` — no OpenAI key or internet access required.

**Services:**

| Service | URL |
|---|---|
| Control Plane API | http://localhost:3000 |
| Dashboard | http://localhost:5173 |
| n8n | http://localhost:5678 (admin / admin123) |
| Postgres | localhost:5432 |

**Optional — real local LLM via Ollama (no OpenAI account, ~2 GB download):**

```bash
docker compose -f infra/docker/docker-compose.test.yml --profile ollama up
# Then pull a model (once):
docker exec easybook-ollama-1 ollama pull llama3.2
```

Set `LLM_PROVIDER=ollama` in `.env.test` and restart.

---

### Option 2 — Bare host (requires Go 1.22 + Node 20)

```bash
# 1. Environment variables
cp .env.example .env

# 2. Start Postgres + n8n
docker compose -f infra/docker/docker-compose.dev.yml up -d

# 3. Run migrations
cd packages/database && npx prisma migrate dev && cd ../..

# 4. Start the control-plane (Go)
cd apps/control-plane
go mod tidy
make dev          # hot-reload with Air

# 5. (In another terminal) start the dashboard
cd apps/dashboard
npm install
npm run dev
```

---

## Testing the Agent Loop

Once the stack is running, simulate WhatsApp messages with curl:

```bash
# Book a court
curl -X POST http://localhost:3000/api/v1/webhooks/whatsapp/twilio \
  -d "From=whatsapp:+5491100000001&Body=Quiero reservar la cancha 1 mañana a las 19hs&ProfileName=Test+User"

# Cancel a booking
curl -X POST http://localhost:3000/api/v1/webhooks/whatsapp/twilio \
  -d "From=whatsapp:+5491100000001&Body=Cancelar mi turno de mañana&ProfileName=Test+User"

# Health check
curl http://localhost:3000/health
```

With `LLM_PROVIDER=mock` the agent loop completes in < 100 ms with no network calls. Watch the full agent tool-call sequence in:

```bash
docker compose -f infra/docker/docker-compose.test.yml logs -f control-plane
```

---

## LLM Provider Options

Controlled by the `LLM_PROVIDER` env var in `.env.test`:

| Value | Description | Requires |
|---|---|---|
| `mock` | Scripted responses, no network (default) | Nothing |
| `ollama` | Real local model via Ollama container | `--profile ollama` + model pull |
| `openai` | Real OpenAI GPT-4o-mini | `OPENAI_API_KEY` |

---

## Documentation

- [Full project context](docs/PROJECT_CONTEXT.md)
- [Architecture Decision Records](docs/ADR/)
- [Operational runbooks](docs/runbooks/)

## License

Private — all rights reserved.
