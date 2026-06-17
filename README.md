# EasyBook 🎾

Plataforma SaaS multi-tenant que automatiza la gestión de turnos de canchas de pádel vía WhatsApp, usando un agente de IA (Claude) orquestado con n8n.

## Stack

| Componente | Tecnología |
|---|---|
| Control Plane API | Go 1.22 + Chi + pgx/v5 |
| Dashboard | React + Vite + TypeScript |
| Orquestación de mensajería | n8n (self-hosted) |
| Base de datos | PostgreSQL (schema-per-tenant + RLS) |
| Agente IA | GPT-4o-mini (OpenAI) |
| Infra | Hetzner VPS + Docker Compose |

## Estructura del repo

```
easybook/
├── apps/
│   ├── control-plane/   # Go — API principal, provisioning, agente IA
│   ├── dashboard/        # React — UI para el club
│   └── n8n-workflows/    # JSON exports de workflows de n8n
├── packages/
│   └── database/         # Prisma schema + migrations (fuente de verdad del esquema SQL)
├── infra/
│   ├── docker/            # docker-compose dev/prod
│   ├── hetzner/            # cloud-init, setup del VPS
│   └── scripts/             # provisioning, utilidades
├── docs/
│   ├── ADR/                  # decisiones de arquitectura
│   └── runbooks/              # guías operativas
└── .github/workflows/          # CI/CD
```

> **Nota de arquitectura:** el schema de la base de datos vive en `packages/database` usando Prisma solo como herramienta de migraciones versionadas (no como ORM en runtime). El `control-plane` en Go usa `sqlc` para generar código type-safe a partir de SQL plano, leyendo el mismo esquema.

## Quickstart local

```bash
# 1. Variables de entorno
cp .env.example .env
# completar OPENAI_API_KEY como mínimo

# 2. Levantar Postgres + n8n
docker compose -f infra/docker/docker-compose.dev.yml up -d

# 3. Correr migraciones
cd packages/database && npx prisma migrate dev && cd ../..

# 4. Arrancar el control-plane (Go)
cd apps/control-plane
go mod tidy
make dev          # hot-reload con Air

# 5. (En otra terminal) arrancar el dashboard
cd apps/dashboard
npm install
npm run dev
```

Servicios disponibles:
- Control Plane API → `http://localhost:3000`
- Dashboard → `http://localhost:5173`
- n8n → `http://localhost:5678` (admin / admin123 en dev)

## Documentación

- [Contexto completo del proyecto](docs/PROYECTO_CONTEXTO.md)
- [Architecture Decision Records](docs/ADR/)
- [Runbooks operativos](docs/runbooks/)

## Licencia

Privado — todos los derechos reservados.
