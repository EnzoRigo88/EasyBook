# EasyBook

Multi-tenant SaaS platform that automates booking management for padel courts via WhatsApp, using an AI agent orchestrated with n8n.

## Stack

| Component | Technology |
|---|---|
| Control Plane API | Go 1.22 + Chi + pgx/v5 |
| Dashboard | React + Vite + TypeScript |
| Messaging Orchestration | n8n (self-hosted) |
| Database | PostgreSQL (schema-per-tenant + RLS) |
| AI Agent | GPT-4o-mini (OpenAI) |
| Infrastructure | Hetzner VPS + Docker Compose |

## Repository Structure

```
easybook/
├── apps/
│   ├── control-plane/   # Go — main API, provisioning, AI agent
│   ├── dashboard/        # React — club management UI
│   └── n8n-workflows/    # n8n workflow JSON exports
├── packages/
│   └── database/         # Prisma schema + migrations (SQL schema source of truth)
├── infra/
│   ├── docker/            # docker-compose dev/prod
│   ├── hetzner/           # cloud-init, VPS setup
│   └── scripts/           # provisioning utilities
├── docs/
│   ├── ADR/               # architecture decision records
│   └── runbooks/          # operational guides
└── .github/workflows/     # CI/CD
```

> **Architecture note:** the database schema lives in `packages/database` using Prisma solely as a versioned migration tool (not as a runtime ORM). The Go `control-plane` uses `sqlc` to generate type-safe code from plain SQL against the same schema.

## Local Quickstart

```bash
# 1. Environment variables
cp .env.example .env
# fill in OPENAI_API_KEY at minimum

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

Available services:
- Control Plane API → `http://localhost:3000`
- Dashboard → `http://localhost:5173`
- n8n → `http://localhost:5678` (admin / admin123 in dev)

## Documentation

- [Full project context](docs/PROJECT_CONTEXT.md)
- [Architecture Decision Records](docs/ADR/)
- [Operational runbooks](docs/runbooks/)

## License

Private — all rights reserved.
