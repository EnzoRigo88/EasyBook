# EasyBook — Project Context

> Context document for use with Claude.ai Projects.  
> Last updated: June 2026 · Author: Engineering Manager

> **Architecture decisions that evolved from initial planning:**
> - The Control Plane was rewritten in **Go** (not Node.js) — see `ADR-001`
> - The AI model is **GPT-4o-mini** (not Claude Sonnet) — see `ADR-002`
>
> Everything else (multi-tenant architecture, roadmap, infrastructure costs, onboarding) remains current.

---

## 1. What We Are Building

A **multi-tenant SaaS platform** that automates booking management for padel courts via WhatsApp (and other channels). Clubs subscribe to the platform and their customers book, cancel, and reschedule court time by chatting with an AI bot — no phone calls, no apps.

**Core problem:** Padel clubs manage bookings through manual WhatsApp messages, phone calls, and group chats. It's chaotic, causes double-booking errors, and costs club owners several hours of operational overhead per week.

**Solution:** A conversational AI agent connected to the club's WhatsApp Business account that understands natural language, checks real-time availability, creates and cancels bookings, and escalates to a human only when necessary.

---

## 2. Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Inbound channel | WhatsApp Business API (Meta Cloud direct) | Cheaper than Twilio at scale. Phase 1 can start with Twilio sandbox. |
| Orchestration | n8n (self-hosted, Docker) | Visual workflows, easy to iterate without code deploys. Runs on Hetzner. |
| AI Agent | GPT-4o-mini (OpenAI API) | Cost-effective, strong Spanish comprehension. |
| Agent memory | PostgreSQL (schema-per-tenant) | Conversation history per WA number. Isolated by Postgres schema. |
| Database | PostgreSQL with Row-Level Security | Single server, multiple schemas (`club_uuid`). RLS enforced by `club_id`. |
| Control Plane API | Go 1.22 + Chi + pgx/v5 | Automatic provisioning of new clubs. Deployed on Hetzner via Docker Compose. |
| Club dashboard | React + Vite + TypeScript | Booking calendar view, metrics, configuration. |
| CDN / Proxy | Cloudflare (Free) | Cache + DDoS protection. |
| Compute infra | Hetzner VPS (CX22 €4.55/mo) | Best price/performance for predictable workloads. |
| Backups | Hetzner Object Storage | Automatic weekly snapshots. €0.50/mo. |
| Monitoring | Grafana Cloud (Free tier) | n8n + DB + API metrics. Alerts if a club's workflow fails. |
| Payments (Phase 3) | Mercado Pago | Payment links via WA. 3.99% + VAT per transaction, no fixed cost. |

---

## 3. Multi-Tenant Architecture

### Strategy: n8n Shard Groups

Not one n8n per club (too expensive) nor one shared instance for everyone (too fragile). The architecture uses shard groups:

```
WhatsApp webhook → Control Plane API
                        ↓
              identifies club by WA number
                        ↓
              routes to correct n8n shard
                        ↓
        ┌───────────────┼───────────────┐
   Shard A (n8n #1)  Shard B (n8n #2)  Shard C (n8n #3)
   clubs 001–025     clubs 026–050      clubs 051–075
        ↓                  ↓                  ↓
   Postgres schema    Postgres schema    Postgres schema
   club_001...025     club_026...050     club_051...075
```

- **1 shard = ~25 clubs = 1 Docker container = €4.55/mo (Hetzner)**
- 1 Hetzner CX32 server can comfortably run 2–3 shards
- The Control Plane assigns new clubs to the lowest-load shard
- A shard failure affects only 25 clubs, not all of them

### Postgres: Schema-per-Tenant with RLS

```sql
-- Each club gets its own schema
CREATE SCHEMA club_a1b2c3;

-- Row-Level Security enabled on all tables
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON bookings
  USING (club_id = current_setting('app.club_id')::uuid);
```

The Control Plane middleware injects `SET app.club_id = 'uuid'` into every connection.

---

## 4. Roadmap by Phase

### Phase 1 — MVP (Weeks 1–3)

**Goal:** Functional WhatsApp bot with real bookings at 1 pilot club.

**Deliverables:**
- WhatsApp webhook (Twilio sandbox → production)
- n8n workflow: intent detection + booking tool + confirmation
- Google Sheets as a transitional DB (easy to swap, zero ops)
- AI agent with per-user conversation memory
- Automatic reminder 2 hours before each booking

**Infra:** Railway (n8n + Postgres)  
**Cost:** ~$20/mo  
**Multi-tenancy:** Single-tenant, 1 hardcoded club

---

### Phase 2 — Alpha · Real Product (Weeks 4–8)

**Goal:** Real database, cancellations, first 5 paying clubs.

**Deliverables:**
- Migration to PostgreSQL with per-club schemas
- Cancellation and rescheduling via WhatsApp
- Automatic waitlist management per time slot
- Control Plane API for automatic club provisioning
- Basic read-only dashboard for clubs (React)
- Onboarding flow: web form → provisioned in < 30 seconds

**Infra:** Hetzner CX22 (Docker Compose, 1 n8n shard) + Railway for Control Plane  
**Cost:** ~$50/mo (5 clubs)  
**Multi-tenancy:** Schema-per-tenant, 1 shard, RLS active

---

### Phase 3 — Beta · Payments + Scale (Weeks 9–14)

**Goal:** Integrated payments, up to 50 clubs, full dashboard.

**Deliverables:**
- Mercado Pago: payment link per booking sent via WhatsApp
- Recurring bookings (e.g., every Friday at 8 PM)
- Full web dashboard: calendar, bookings, revenue, metrics
- Self-service portal for club configuration
- Improved human escalation: Slack tickets to club team
- Additional channels: Instagram DM + Telegram (optional)
- Shard groups: multiple n8n containers, routing via Control Plane

**Infra:** Hetzner CX32 (2–3 shards) + Fly.io (Control Plane + Dashboard)  
**Cost:** ~$100–150/mo base (50 clubs)

---

### Phase 4 — v1.0 · Full SaaS (Weeks 15+)

**Goal:** SaaS product with pricing plans, 100+ clubs, auto-provisioning.

**Deliverables:**
- Pricing plans: Starter ($29) / Pro ($59) / Enterprise ($99+)
- Stripe billing for club subscriptions
- White-label: clubs can customize bot name and logo
- Advanced analytics: demand by time slot, no-show rate
- SLA monitoring: alerts if a workflow fails for any club
- Multi-region if there is traction in Brazil/US

**Infra:** Hetzner Cloud multi-server + K3s + Fly.io global  
**Cost:** ~$200–300/mo base (100+ clubs)

---

## 5. Cost Model

### Key Variables

- **Infra cost per n8n shard:** €4.55/mo → ~$5 → $0.20/club/mo amortized
- **WhatsApp (Meta Cloud, AR):** ~$0.04/conversation (1,000 free conversations/mo total)
- **GPT-4o-mini:** ~$0.008/conversation (2,300 tokens input + 500 output)
- **Control Plane API:** $5/mo fixed (Railway)

### Total Cost at 10 Clubs × 120 Conversations/Month

| Component | Cost/mo |
|-----------|---------|
| n8n shard (Hetzner) | $5.00 |
| Control Plane (Railway) | $5.00 |
| WhatsApp Meta (1,200 convs − 1,000 free) | $8.00 |
| OpenAI API (1,200 convs) | $9.60 |
| Object Storage + extras | $0.50 |
| **Total** | **~$28/mo** |
| **Per club** | **~$2.80/mo** |

### Suggested Pricing

| Plan | Price | Estimated Margin |
|------|-------|-----------------|
| Starter | $29/mo | ~90% |
| Pro | $59/mo | ~95% |
| Enterprise | $99+/mo | ~97% |

---

## 6. Repository Structure

**Monorepo with pnpm workspaces + Turborepo.**

```
easybook/
├── apps/
│   ├── control-plane/      # Go + Chi — main API + provisioning
│   ├── dashboard/          # React + Vite — club UI
│   └── n8n-workflows/      # n8n workflow JSON exports + sync scripts
├── packages/
│   ├── database/           # Prisma schema + migrations
│   └── shared-types/       # Shared TypeScript interfaces
├── infra/
│   ├── docker/             # docker-compose.dev.yml + docker-compose.prod.yml
│   ├── scripts/            # provision-club.sh, shard-rebalance.sh
│   └── hetzner/            # cloud-init, server setup scripts
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # PR checks (lint + test + typecheck) — ~2.5 min
│   │   ├── deploy-staging.yml  # triggered on push to develop
│   │   ├── deploy-prod.yml     # triggered on push to main (requires review)
│   │   └── n8n-sync.yml        # automatic workflow sync to shards
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── CODEOWNERS
├── docs/
│   ├── ADR/                # Architecture Decision Records
│   └── runbooks/           # add-club, rotate-keys, shard-ops
├── turbo.json
├── .env.example
└── package.json            # root workspace
```

### Repository Architecture Decisions

- **Monorepo:** atomic changes across `control-plane` and `packages/database` in a single PR
- **Turborepo:** intelligent build cache, parallel lint/test/build pipelines
- **Docker for prod:** each app has its own `Dockerfile`, orchestrated with `docker-compose.prod.yml`
- **GitHub Secrets:** never commit credentials — `DATABASE_URL`, `OPENAI_API_KEY`, `TWILIO_TOKEN` live only in Secrets

---

## 7. Development Standards

### Go (Control Plane)
- `sqlc` for type-safe database access — no raw string interpolation
- Middleware injects `app.club_id` into every Postgres connection for RLS enforcement
- Structured logging with `slog` — always include `club_id` and `correlation_id` in context

### TypeScript (Dashboard)
- `strict: true` in all `tsconfig.json` — no implicit `any`
- Zod for runtime validation of external inputs (WA webhooks, onboarding forms)
- Explicit return types on all public service functions

### Git & Commits
- **Conventional Commits:** `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`
- **Branch naming:** `feat/booking-cancellation`, `fix/wa-webhook-timeout`
- No direct pushes to `main` or `develop` — always via PR with at least 1 review
- Branch protection rules active on both branches

### Testing
- Unit tests with **Vitest** (TypeScript) / `go test` (Go) for pure business logic
- Integration tests for critical endpoints: create booking, cancel booking, provision club
- Minimum **70%** coverage on `packages/database` and service layers
- Test observable behavior, not implementation details

### API Design
- Versioned REST: `/api/v1/clubs/:clubId/bookings`
- Tenant middleware: injects Postgres schema from JWT on every request
- Consistent error shape: `{ error: string, code: string, details?: any }`
- Rate limiting per IP and per `clubId` on public endpoints

### Security
- Webhook signature validation for all providers (Meta WA, Twilio, Mercado Pago)
- Short-lived JWTs (1h) + rotated refresh tokens
- Database access only via `sqlc`-generated queries — never manual SQL string interpolation
- Secrets only in `.env` locally and GitHub Secrets in CI — never committed to the repo

### Observability
- Structured logging (JSON) — always include `clubId` and `correlationId` in log context
- n8n metrics exported to Grafana Cloud (free up to 10k series)
- Alert if a club's workflow has not executed in > 24 hours
- OpenTelemetry tracing on critical booking flows

---

## 8. Club Onboarding

The process is fully automated. No team member should need to touch anything to activate a new club.

### Flow

**Step 1 — Form (5–10 min for the club)**  
The club fills out a web form with: name, WhatsApp Business number, courts (name, type), operating hours per weekday, price per hour, admin email.

**Step 2 — Automatic provisioning (< 30 seconds)**  
The Control Plane API automatically:
- Creates a Postgres schema: `club_{uuid}`
- Seeds tables: `courts`, `schedules`, `pricing`, `users`
- Copies the n8n workflow template and parametrizes it with `club_id`
- Activates the WhatsApp webhook for the club's number
- Assigns the club to the lowest-load n8n shard
- Generates dashboard credentials (admin user)

**Step 3 — Sandbox (48–72 hours)**  
- The bot is active but bookings are flagged as `is_sandbox=true`
- The club's team tests booking conversations
- QA checklist must be approved before go-live

**Step 4 — Go-live (1 click)**  
- Toggle `is_sandbox=false` via Control Plane
- Automatic "Welcome Kit" email: WA QR code, dashboard tutorial, support contact
- Follow-up check-in at 24 hours

**Step 5 — Continuous monitoring**  
- Alert if a club's booking error rate exceeds 5%
- Automated NPS survey at 7 and 30 days post-go-live
- Upgrade prompts when approaching plan limits

### Onboarding Success KPIs
- Form to sandbox bot: < 10 minutes
- Automatic provisioning: < 30 seconds
- Clubs reaching go-live within 7 days: > 80%
- Onboarding NPS: > 4.5/5
- Human support time per activation: < 2 hours

---

## 9. Bot Conversation Flows

### New Booking
```
User → "I want court 2 tomorrow at 7 PM"
n8n trigger → AI Agent (GPT-4o-mini)
Agent parses: date, time, court, user (by WA number)
Tool: check_availability(court=2, date=tomorrow, time=19:00)
Postgres responds: available ✓
Tool: create_booking(...)
Confirmation via WA + logged in Postgres
Automatic reminder at t-2h
```

### Cancellation
```
User → "Cancel my booking for tomorrow"
Agent identifies user's active booking (by WA number)
Tool: cancel_booking(booking_id)
Postgres: releases the slot + notifies first user on waitlist
Confirmation sent to the user who cancelled
```

### Human Escalation
```
User → complaint, or > 3 failed booking attempts
Agent classifies severity
Tool: create_support_ticket(severity, summary)
Club notified via Slack/email with summary
Human agent takes over the WhatsApp thread directly
Resolution recorded in DB
```

---

## 10. Required Environment Variables

```bash
# Control Plane API
DATABASE_URL=postgresql://user:pass@host:5432/easybook
OPENAI_API_KEY=sk-...
JWT_SECRET=...
JWT_REFRESH_SECRET=...

# WhatsApp
META_WA_PHONE_NUMBER_ID=...
META_WA_ACCESS_TOKEN=...
META_WA_WEBHOOK_VERIFY_TOKEN=...
# Phase 1 alternative:
TWILIO_ACCOUNT_SID=...
TWILIO_AUTH_TOKEN=...
TWILIO_WA_NUMBER=whatsapp:+...

# n8n
N8N_BASIC_AUTH_USER=admin
N8N_BASIC_AUTH_PASSWORD=...
N8N_ENCRYPTION_KEY=...
N8N_WEBHOOK_URL=https://n8n.yourdomain.com

# Mercado Pago (Phase 3)
MP_ACCESS_TOKEN=...
MP_WEBHOOK_SECRET=...

# Infrastructure
HETZNER_TOKEN=...       # for server provisioning scripts
SENTRY_DSN=...          # error tracking (optional)
```

---

## 11. Development Environment

1. **Claude.ai Projects** (this context) — code generation, architecture decisions
2. **GitHub Codespaces** — browser-based VS Code, works on iPad Safari
3. **Claude Code CLI** installed in the Codespace: `npm i -g @anthropic-ai/claude-code`
4. **Hetzner VPS** for staging (same server that runs n8n in production)

Typical workflow:
- Architecture and code generated in Claude.ai → pasted into Codespace
- Claude Code CLI for autonomous refinement (tests, bug fixes, refactors)
- `git push → CI → automatic deploy to staging` in ~5 minutes

---

## 12. Open Decisions (ADRs to Write)

These decisions are not finalized and require a formal ADR before implementation:

| # | Decision | Options | Status |
|---|---------|---------|--------|
| 001 | WA Provider Phase 1 | Twilio vs Meta direct | ⏳ Pending |
| 002 | AI Model | GPT-4o-mini vs Claude Sonnet 4.6 | ⏳ Pending |
| 003 | Payments MVP | Mercado Pago from Phase 1 vs defer to Phase 3 | ⏳ Pending — **recommendation: defer** |
| 004 | Dashboard Phase 2 | Retool (fast, $) vs custom React | ⏳ Pending |
| 005 | Multi-region | Single-region (AR) vs multi-region from Phase 3 | ⏳ Pending |

---

## 13. Out of Scope

- Mobile app for players (could be Phase 4+, not before)
- Tournament or league management
- On-site hardware (court sensors, access control)
- Member or membership management (a separate product)
- ERP or accounting system integrations
