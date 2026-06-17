# Paddle SaaS — Contexto del Proyecto

> Documento de contexto para el proyecto en Claude.ai.  
> Última actualización: Junio 2026 · Autor: Engineering Manager

> ⚠️ **Nota:** algunas decisiones evolucionaron y están formalizadas en `docs/ADR/`:
> - El Control Plane se reescribió en **Go** (no Node.js) — ver `ADR-001`
> - El modelo de IA es **GPT-4o-mini** (no Claude) — ver `ADR-002`
>
> El resto del contenido (arquitectura multi-tenant, roadmap, costos de infra, onboarding) sigue vigente.

---

## 1. Qué estamos construyendo

Una **plataforma SaaS multi-tenant** que automatiza la gestión de turnos de canchas de pádel vía WhatsApp (y otros canales). Los clubes se suscriben a la plataforma y sus clientes reservan, cancelan y reprograman turnos conversando con un bot de IA, sin llamar ni entrar a ninguna app.

**Problema central:** Los clubes de pádel gestionan turnos por WhatsApp manual, llamadas y grupos de chat. Es caótico, genera errores de doble-booking, y el dueño del club pierde horas operativas por semana.

**Solución:** Un agente conversacional IA conectado al WhatsApp Business del club que entiende lenguaje natural, consulta disponibilidad en tiempo real, crea/cancela reservas y escala a un humano solo cuando es necesario.

---

## 2. Stack tecnológico

| Capa | Tecnología | Justificación |
|------|-----------|--------------|
| Canal entrada | WhatsApp Business API (Meta Cloud directa) | Más barato que Twilio a escala. Fase 1 puede arrancar con Twilio sandbox. |
| Orquestación | n8n (self-hosted, Docker) | Workflows visuales, fácil de iterar sin deploy de código. Corre en Hetzner. |
| Agente IA | Claude Sonnet (Anthropic API) | Mejor manejo de español y contexto largo. `$3/M tokens input, $15/M output`. |
| Memoria agente | PostgreSQL (schema-per-tenant) | Historial de conversación por número de WA. Aislamiento por schema de Postgres. |
| Base de datos | PostgreSQL con Row-Level Security | Un server, múltiples schemas (`club_uuid`). RLS activa por `club_id`. |
| Control Plane API | Node.js + Fastify | Provisioning automático de nuevos clubes. Deploy en Railway (Fase 1) o Fly.io (Fase 2+). |
| Dashboard club | React + Vite | Vista del calendario de reservas, métricas, configuración. Fly.io. |
| CDN / Proxy | Cloudflare (Free) | Caché + DDoS protection. |
| Infra cómputo | Hetzner VPS (CX22 €4.55/mes) | Mejor precio/performance del mercado para cargas predecibles. |
| Backups | Hetzner Object Storage | Snapshots automáticos semanales. €0.50/mes. |
| Monitoreo | Grafana Cloud (Free tier) | Métricas de n8n + DB + API. Alertas si workflow de un club falla. |
| Pagos (Fase 3) | Mercado Pago | Links de pago por WA. 3.99% + IVA por transacción, sin costo fijo. |

---

## 3. Arquitectura multi-tenant

### Estrategia: Shard groups de n8n

No hay un n8n por club (muy caro) ni uno solo para todos (muy frágil). La arquitectura es:

```
WhatsApp webhook → Control Plane API
                        ↓
              identifica club por número WA
                        ↓
              enruta al shard de n8n correcto
                        ↓
        ┌───────────────┼───────────────┐
   Shard A (n8n #1)  Shard B (n8n #2)  Shard C (n8n #3)
   clubes 001–025    clubes 026–050     clubes 051–075
        ↓                  ↓                  ↓
   Postgres schema    Postgres schema    Postgres schema
   club_001...025     club_026...050     club_051...075
```

- **1 shard = ~25 clubes = 1 contenedor Docker = €4.55/mes (Hetzner)**
- 1 servidor Hetzner CX32 puede correr 2–3 shards cómodamente
- El Control Plane decide a qué shard va cada club nuevo (el de menor carga)
- Fallo de un shard afecta solo 25 clubes, no a todos

### Postgres: schema-per-tenant con RLS

```sql
-- Cada club tiene su propio schema
CREATE SCHEMA club_a1b2c3;

-- Row-Level Security activo en todas las tablas
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON bookings
  USING (club_id = current_setting('app.club_id')::uuid);
```

El middleware del Control Plane inyecta `SET app.club_id = 'uuid'` en cada conexión.

---

## 4. Roadmap por fases

### Fase 1 — MVP (Semanas 1–3)

**Objetivo:** Bot de WhatsApp funcional con reservas reales en 1 club piloto.

**Entregables:**
- Webhook WhatsApp (Twilio sandbox → producción)
- n8n workflow: intent detection + booking tool + confirmación
- Google Sheets como DB transitoria (fácil de cambiar, sin ops)
- Agente Claude con memoria de conversación por usuario
- Reminder automático 2 horas antes del turno

**Infra:** Railway (n8n + Postgres)  
**Costo:** ~$20/mes  
**Multi-tenant:** Single-tenant, 1 club hardcodeado  

---

### Fase 2 — Alpha · Producto Real (Semanas 4–8)

**Objetivo:** Base de datos real, cancelaciones, primeros 5 clubes pagantes.

**Entregables:**
- Migración a PostgreSQL con schemas por club
- Cancelación y reprogramación vía WA
- Lista de espera automática por horario
- Control Plane API para provisioning automático de clubes
- Dashboard básico read-only para el club (React)
- Onboarding flow: formulario web → provisioning en <30 segundos

**Infra:** Hetzner CX22 (Docker Compose, 1 shard n8n) + Railway para Control Plane  
**Costo:** ~$50/mes (5 clubes)  
**Multi-tenant:** Schema-per-tenant, 1 shard, RLS activo  

---

### Fase 3 — Beta · Pagos + Escala (Semanas 9–14)

**Objetivo:** Pagos integrados, hasta 50 clubes, dashboard completo.

**Entregables:**
- Mercado Pago: link de pago en WA por cada reserva
- Reservas recurrentes (ej: viernes 20hs todas las semanas)
- Dashboard web completo: calendario, reservas, ingresos, métricas
- Portal de self-service para configuración del club
- Escalado humano mejorado: tickets con Slack del club
- Canales adicionales: Instagram DM + Telegram (opcionales)
- Shard groups: múltiples n8n containers, routing por Control Plane

**Infra:** Hetzner CX32 (2–3 shards) + Fly.io (Control Plane + Dashboard)  
**Costo:** ~$100–150/mes base (50 clubes)  

---

### Fase 4 — v1.0 · SaaS Completo (Semanas 15+)

**Objetivo:** Producto SaaS con pricing plans, 100+ clubes, auto-provisioning.

**Entregables:**
- Pricing plans: Starter ($29) / Pro ($59) / Enterprise ($99+)
- Billing con Stripe para subscripciones de clubes
- White-label: el club puede customizar nombre y logo del bot
- Analytics avanzado: demanda por horario, tasa de no-show
- SLA monitoring: alertas si un workflow falla para un club
- Multi-region si hay tracción en Brasil/US

**Infra:** Hetzner Cloud multi-server + K3s + Fly.io global  
**Costo:** ~$200–300/mes base (100+ clubes)  

---

## 5. Modelo de costos

### Variables clave

- **Costo infra por shard n8n:** €4.55/mes → ~$5 → $0.20/club/mes amortizado
- **WhatsApp (Meta Cloud, AR):** ~$0.04/conversación (1000 convs gratis/mes totales)
- **Claude Sonnet:** ~$0.013/conversación (2300 tokens input + 500 output)
- **Control Plane API:** $5/mes fijo (Railway)

### Costo total a 10 clubes × 120 conv/mes

| Componente | Costo/mes |
|-----------|-----------|
| n8n shard (Hetzner) | $5.00 |
| Control Plane (Railway) | $5.00 |
| WhatsApp Meta (1200 convs − 1000 free) | $8.00 |
| Claude API (1200 convs) | $15.60 |
| Object Storage + extras | $0.50 |
| **Total** | **~$34/mes** |
| **Por club** | **~$3.40/mes** |

### Pricing sugerido

| Plan | Precio | Margen estimado |
|------|--------|----------------|
| Starter | $29/mes | ~70% |
| Pro | $59/mes | ~75% |
| Enterprise | $99+/mes | ~80% |

---

## 6. Estructura del repositorio

**Monorepo con pnpm workspaces + Turborepo.**

```
paddle-saas/
├── apps/
│   ├── control-plane/      # Node.js + Fastify — API principal + provisioning
│   ├── dashboard/          # React + Vite — UI para el club
│   └── n8n-workflows/      # JSON exports de workflows de n8n + scripts de sync
├── packages/
│   ├── database/           # Prisma schema + migrations
│   ├── shared-types/       # Interfaces TypeScript compartidas
│   └── ui/                 # Componentes React reutilizables
├── infra/
│   ├── docker/             # docker-compose.dev.yml + docker-compose.prod.yml
│   ├── scripts/            # provision-club.sh, shard-rebalance.sh
│   └── hetzner/            # cloud-init, server setup scripts
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # PR checks (lint + test + typecheck) — ~2.5 min
│   │   ├── deploy-staging.yml  # → push a develop
│   │   ├── deploy-prod.yml     # → push a main (requiere review)
│   │   └── n8n-sync.yml        # sync automático de workflows a shards
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── CODEOWNERS
├── docs/
│   ├── ADR/                # Architecture Decision Records
│   └── runbooks/           # add-club, rotate-keys, shard-ops
├── turbo.json
├── .env.example
└── package.json            # root workspace
```

### Decisiones de arquitectura del repo

- **Monorepo:** cambios atómicos entre `control-plane` y `packages/database` en un solo PR
- **Turborepo:** cache inteligente de builds, paralelización de lint/test/build
- **Docker para prod:** cada app tiene su `Dockerfile`, orquestado con `docker-compose.prod.yml`
- **GitHub Secrets:** nunca commitear credenciales — `DATABASE_URL`, `ANTHROPIC_API_KEY`, `TWILIO_TOKEN` solo en Secrets

---

## 7. Estándares de desarrollo

### TypeScript
- `strict: true` en todos los `tsconfig.json` — cero `any` implícito
- Zod para validación en runtime de inputs externos (webhooks de WA, formularios de onboarding)
- Return types explícitos en todas las funciones públicas de servicios

### Git & Commits
- **Conventional Commits:** `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`
- **Branch naming:** `feat/booking-cancellation`, `fix/wa-webhook-timeout`
- No push directo a `main` ni `develop` — siempre vía PR con mínimo 1 review
- Branch protection rules activas en ambas ramas

### Testing
- Unit tests con **Vitest** para lógica de negocio pura (disponibilidad, detección de conflictos)
- Integration tests para endpoints críticos: crear reserva, cancelar, provisionar club
- Coverage mínimo **70%** en `packages/database` y `services/`
- Testear comportamiento observable, no implementación

### API Design
- REST versionado: `/api/v1/clubs/:clubId/bookings`
- Middleware de tenant: inyecta schema de Postgres según JWT en cada request
- Errores consistentes: `{ error: string, code: string, details?: any }`
- Rate limiting por IP y por `clubId` en endpoints públicos

### Seguridad
- Webhook signature validation en todos los providers (Meta WA, Twilio, Mercado Pago)
- JWT con expiración corta (1h) + refresh tokens rotados
- SQL solo vía Prisma — jamás interpolación manual de strings SQL
- Secrets únicamente en `.env` local y GitHub Secrets en CI — nunca en el repo

### Observabilidad
- Structured logging con **Pino** (JSON) — siempre incluir `clubId` y `correlationId` en contexto
- Métricas de n8n exportadas a Grafana Cloud (free hasta 10k series)
- Alert si el workflow de un club no ejecuta en >24hs
- OpenTelemetry para tracing de flows críticos de reserva

---

## 8. Onboarding de clubes

El proceso es completamente automatizado. Un humano del equipo no debería tocar nada para dar de alta un club nuevo.

### Flujo

**Paso 1 — Formulario (5–10 min para el club)**
El club completa un formulario web con: nombre, número WhatsApp Business, canchas (nombre, tipo), horarios de operación por día de semana, precio por hora, email del admin.

**Paso 2 — Provisioning automático (<30 segundos)**
El Control Plane API ejecuta automáticamente:
- Crea schema en Postgres: `club_{uuid}`
- Seed de tablas: `courts`, `schedules`, `pricing`, `users`
- Copia el template de n8n workflow y lo parametriza con `club_id`
- Activa webhook de WhatsApp para el número del club
- Asigna al shard de n8n con menor carga
- Genera credenciales del dashboard (usuario admin)

**Paso 3 — Sandbox (48–72hs)**
- El bot está activo pero reservas marcadas como `is_sandbox=true`
- El equipo del club prueba conversaciones de reserva
- Checklist de QA que debe aprobar antes del go-live

**Paso 4 — Go-live (1 click)**
- Toggle de `is_sandbox=false` vía Control Plane
- Envío automático de "Kit de bienvenida" por email: QR del WA, tutorial del dashboard, contacto de soporte
- Check-in de seguimiento a las 24hs

**Paso 5 — Monitoreo continuo**
- Alerta si tasa de error de reservas del club supera el 5%
- NPS automático a los 7 y 30 días post-go-live
- Upgrade prompts cuando se acercan al límite de su plan

### KPIs de éxito del onboarding
- Formulario a bot en sandbox: < 10 minutos
- Provisioning automático: < 30 segundos
- Clubes que llegan al go-live en < 7 días: > 80%
- NPS del proceso: > 4.5/5
- Tiempo de soporte humano por alta: < 2 horas

---

## 9. Flujos conversacionales del bot

### Reserva nueva
```
Usuario → "Quiero la cancha 2 mañana a las 19hs"
n8n trigger → AI Agent (Claude)
Claude parsea: fecha, hora, cancha, usuario (por número WA)
Tool: check_availability(court=2, date=mañana, time=19:00)
Postgres responde: disponible ✓
Tool: create_booking(...)
Confirmación por WA + log en Google Sheets (Fase 1) / Postgres (Fase 2+)
Reminder automático en t-2hs
```

### Cancelación
```
Usuario → "Cancelo mi turno de mañana"
Claude identifica reserva activa del usuario (por número WA)
Tool: cancel_booking(booking_id)
Postgres: libera el slot + notifica al primer usuario en lista de espera
Confirmación al usuario que canceló
```

### Escalado a humano
```
Usuario → problema, queja, o >3 intentos fallidos de reserva
Claude clasifica severidad
Tool: create_support_ticket(severity, summary)
Notificación al club vía Slack/email con el resumen
Agente humano toma el hilo directamente en WhatsApp
Resolución registrada en DB
```

---

## 10. Variables de entorno requeridas

```bash
# Control Plane API
DATABASE_URL=postgresql://user:pass@host:5432/paddle_saas
ANTHROPIC_API_KEY=sk-ant-...
JWT_SECRET=...
JWT_REFRESH_SECRET=...

# WhatsApp
META_WA_PHONE_NUMBER_ID=...
META_WA_ACCESS_TOKEN=...
META_WA_WEBHOOK_VERIFY_TOKEN=...
# Alternativa Fase 1:
TWILIO_ACCOUNT_SID=...
TWILIO_AUTH_TOKEN=...
TWILIO_WA_NUMBER=whatsapp:+...

# n8n
N8N_BASIC_AUTH_USER=admin
N8N_BASIC_AUTH_PASSWORD=...
N8N_ENCRYPTION_KEY=...
N8N_WEBHOOK_URL=https://n8n.tudominio.com

# Mercado Pago (Fase 3)
MP_ACCESS_TOKEN=...
MP_WEBHOOK_SECRET=...

# Infra
HETZNER_TOKEN=...       # para scripts de provisioning de servidores
SENTRY_DSN=...          # error tracking (opcional)
```

---

## 11. Ambiente de desarrollo (desde iPad)

El proyecto se desarrolla con el siguiente flujo:

1. **Claude.ai Projects** (este contexto) — generación de código, decisiones de arquitectura
2. **GitHub Codespaces** — VS Code en el browser, funciona en iPad Safari
3. **Claude Code CLI** instalado en el Codespace: `npm i -g @anthropic-ai/claude-code`
4. **Hetzner VPS** para staging (mismo server que corre n8n en producción)

El flujo típico de trabajo:
- Arquitectura y código generado en Claude.ai → pegado en Codespace
- Claude Code CLI para refinamiento autónomo (tests, fix de bugs, refactors)
- `git push → CI → deploy automático a staging` en ~5 minutos

---

## 12. Decisiones pendientes (ADRs a escribir)

Estas decisiones no están cerradas y requieren ADR formal antes de implementar:

| # | Decisión | Opciones | Estado |
|---|---------|---------|--------|
| 001 | Proveedor WA Fase 1 | Twilio vs Meta directo | ⏳ Pendiente |
| 002 | Modelo IA | Claude Sonnet 4.6 vs GPT-4o | ⏳ Pendiente |
| 003 | Pagos MVP | Mercado Pago desde Fase 1 vs diferir a Fase 3 | ⏳ Pendiente — **recomendación: diferir** |
| 004 | Dashboard Fase 2 | Retool (rápido, $) vs React custom | ⏳ Pendiente |
| 005 | Multi-region | Single-region (AR) vs multi-region desde Fase 3 | ⏳ Pendiente |

---

## 13. Lo que NO es este proyecto (scope out)

- App móvil para jugadores (puede ser Fase 4+, no antes)
- Gestión de torneos o ligas
- Hardware en el club (sensores de cancha, control de acceso)
- Gestión de socios / membresías (es otro producto)
- Integración con sistemas de ERP o contabilidad
