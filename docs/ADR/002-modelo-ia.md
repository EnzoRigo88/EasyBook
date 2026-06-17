# ADR-002: Modelo de IA — GPT-4o-mini (no Claude, no Gemini)

**Estado:** Aceptado
**Fecha:** 2026-06-17
**Reemplaza:** decisión pendiente #002 en `docs/PROYECTO_CONTEXTO.md`

## Contexto

El agente conversacional necesita: tool-use confiable (4 herramientas: `check_availability`, `create_booking`, `cancel_booking`, `escalate_to_human`), buen manejo de español rioplatense, y costo mínimo dado que el volumen de conversaciones por club es alto en relación al ticket que paga cada club ($29-99/mes).

## Comparación de costos (junio 2026)

Estimado a 1200 conversaciones/mes (~2300 tokens input + 500 output por conversación):

| Modelo | Input /M tokens | Output /M tokens | Costo mensual estimado |
|---|---|---|---|
| Claude Sonnet 4.6 | $3.00 | $15.00 | ~$17.30 |
| GPT-4o-mini | $0.15 | $0.60 | ~$0.77 |
| Gemini 2.5 Flash-Lite | $0.10 | $0.40 | ~$0.52 |
| Gemini 2.5 Flash | $0.30 | $2.50 | ~$2.33 |

## Decisión

Usar **GPT-4o-mini** vía la API de OpenAI.

Es ~95% más barato que Claude Sonnet a este volumen, y su soporte de function calling es el más maduro y probado del mercado — para un bot donde la confiabilidad de la llamada a `create_booking` importa más que la creatividad de la respuesta, eso pesa más que ahorrar otro $0.25/mes con Gemini Flash-Lite.

SDK usado: `github.com/sashabaranov/go-openai` (third-party, pero el más estable y ampliamente adoptado en el ecosistema Go).

## Alternativas consideradas

- **Claude Sonnet 4.6:** mejor manejo de contexto largo y de matices del español, pero ~22x más caro a este volumen sin una ventaja de calidad que justifique el costo para un caso de uso de tool-use simple y repetitivo.
- **Gemini 2.5 Flash-Lite:** el más barato en papel, pero su posicionamiento público es para *"resúmenes, clasificación, extracción y Q&A simple"* — no hay evidencia clara de robustez en function-calling multi-step (el agentic loop de este bot encadena varias herramientas en secuencia).
- **Gemini 2.5 Flash:** tier intermedio de Google, más caro que GPT-4o-mini sin una ventaja clara para este caso de uso.

## Consecuencias

**Más fácil:**
- El costo de IA pasa de ser ~50% del costo total de infra a ser prácticamente despreciable (<5%)
- Deja margen para subir el volumen de conversaciones por club sin re-evaluar el pricing

**Más difícil / a vigilar:**
- GPT-4o-mini es un modelo más pequeño — monitorear en producción si hay casos de ambigüedad en fechas/horarios que maneje peor que un modelo más grande
- Si en Fase 3+ se necesita razonamiento más complejo (ej: optimización de horarios, recomendaciones), puede ser necesario escalar a GPT-4o o GPT-5-mini para esos flujos específicos, manteniendo GPT-4o-mini para el flujo conversacional base
- Vendor lock-in leve a OpenAI — mitigado porque `AgentService` ya está aislado como un service con interfaz clara, migrar a otro proveedor implica reescribir un solo archivo
