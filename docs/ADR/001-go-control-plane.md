# ADR-001: Control Plane en Go (no Node.js)

**Estado:** Aceptado
**Fecha:** 2026-06-17

## Contexto

El control-plane es el servicio más crítico: recibe webhooks de WhatsApp, orquesta llamadas a la API de Claude, y maneja el provisioning de nuevos clubes. Necesita baja latencia, bajo consumo de memoria (para maximizar clubes por shard de Hetzner), y un binario simple de deployar.

## Decisión

Reescribir el control-plane en **Go 1.22** en vez de Node.js/Fastify.

Stack resultante:
- HTTP router: **Chi** (compatible con `net/http` estándar, sin abstracciones extra)
- Acceso a DB: **sqlc** + **pgx/v5** (SQL plano → código Go type-safe, sin ORM en runtime)
- Logging: **zerolog** (JSON estructurado)
- SDK del modelo de IA: ver `docs/ADR/002-modelo-ia.md` (decisión separada del proveedor de IA)

El dashboard se mantiene en **React + Vite + TypeScript** — no hay alternativa mejor para UI, y Go no compite en ese espacio.

## Alternativas consideradas

- **Node.js + Fastify + Prisma:** Más rápido de arrancar, pero ~10x más RAM por instancia y bundles de ~300MB en Docker vs ~15MB de Go.
- **Node.js + Bun:** Mejora el runtime pero no resuelve el problema de footprint de memoria en contenedores.

## Consecuencias

**Más fácil:**
- Más clubes por shard de Hetzner (mismo €4.55/mes corre más carga)
- Deploy de un solo binario estático, sin `node_modules`
- Concurrencia nativa con goroutines para manejar múltiples webhooks en paralelo

**Más difícil:**
- Curva de aprendizaje de Go para el equipo (mitigado: stdlib es simple, sin magia de frameworks)
- Sin ORM — hay que escribir SQL a mano (mitigado: sqlc genera los tipos, reduce el riesgo de errores)
- Menos librerías del ecosistema que Node para casos muy específicos
