# Runbook: Alta de un nuevo club

> Estado: manual en Fase 1. Se automatiza en Fase 2 vía Control Plane API (ver sección 8 de PROYECTO_CONTEXTO.md).

## Pasos (Fase 1 — manual)

1. Insertar el club en la tabla `clubs` (vía `psql` o script):
   ```sql
   INSERT INTO clubs (name, wa_phone_number, email, is_sandbox)
   VALUES ('Club Piloto', '+5491112345678', 'admin@clubpiloto.com', true);
   ```
2. Insertar las canchas (`courts`), horarios (`schedules`) y precios (`pricing`) asociados.
3. Configurar el webhook de Twilio/Meta para apuntar al número del club.
4. Probar el flujo completo con un mensaje de WhatsApp real.
5. Cuando el club aprueba el sandbox: `UPDATE clubs SET is_sandbox = false WHERE id = '...'`.

## Pasos (Fase 2+ — automatizado)

Ver `internal/service/provisioning.go` en `apps/control-plane` (pendiente de implementar).
