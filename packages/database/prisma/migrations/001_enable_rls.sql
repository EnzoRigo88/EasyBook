-- 001_enable_rls.sql
-- Habilita Row-Level Security en todas las tablas multi-tenant.
-- Correr DESPUÉS de `prisma migrate dev` (Prisma no soporta RLS nativamente).
--
-- Uso:
--   psql $DATABASE_URL -f packages/database/prisma/migrations/001_enable_rls.sql

ALTER TABLE clubs ENABLE ROW LEVEL SECURITY;
ALTER TABLE courts ENABLE ROW LEVEL SECURITY;
ALTER TABLE schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing ENABLE ROW LEVEL SECURITY;
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE waitlist ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversation_messages ENABLE ROW LEVEL SECURITY;

-- Policy: cada tabla solo expone filas donde club_id coincide
-- con la variable de sesión seteada por el middleware (SET app.club_id = '...')

CREATE POLICY tenant_isolation_clubs ON clubs
  USING (id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_courts ON courts
  USING (club_id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_schedules ON schedules
  USING (club_id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_pricing ON pricing
  USING (club_id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_bookings ON bookings
  USING (club_id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_waitlist ON waitlist
  USING (club_id::text = current_setting('app.club_id', true));

CREATE POLICY tenant_isolation_conversation_messages ON conversation_messages
  USING (club_id::text = current_setting('app.club_id', true));

-- IMPORTANTE: el usuario de la app NO debe ser superuser/owner de la tabla,
-- porque RLS no aplica a superusers ni a BYPASSRLS roles.
-- Verificar con: \du   (en psql) que el rol de la app no tenga esos flags.
