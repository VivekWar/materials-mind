#!/bin/bash
# =============================================================================
# Production Migration: Local PostgreSQL → Neon Serverless
# =============================================================================
# Usage: cd /home/vivek/Materials_Mind && bash data/migrate_to_neon.sh
# =============================================================================

set -euo pipefail

# ── Read DATABASE_URL directly from .env (avoid xargs splitting on & chars) ───
if [ -f ".env" ]; then
  NEON_URL=$(grep '^DATABASE_URL=' .env | cut -d'=' -f2-)
fi

if [ -z "${NEON_URL:-}" ]; then
  # Try env var directly
  NEON_URL="${DATABASE_URL:-}"
fi

if [ -z "${NEON_URL:-}" ]; then
  echo "❌ Could not read DATABASE_URL from .env or environment"
  exit 1
fi

echo "✅ Neon URL loaded (host: $(echo "$NEON_URL" | grep -oP '@[^/]+' | tr -d '@'))"
echo ""

# ── Helper: strip \restrict lines and import ──────────────────────────────────
strip_and_import() {
  local FILE="$1"
  local TABLE="$2"
  echo "  → Importing $TABLE from $FILE ..."
  grep -v '^\\restrict' "$FILE" \
    | grep -v '^SET row_security' \
    | psql "$NEON_URL" --set ON_ERROR_STOP=1 -q 2>&1
  echo "  ✅ $TABLE done"
}

# ── STEP 1: Apply schema ───────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 1/4 — Applying schema to Neon..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
grep -v '^\\restrict' data/schema.sql \
  | grep -v '^SET row_security' \
  | psql "$NEON_URL" -q 2>&1 | grep -v '^NOTICE' | grep -v '^$' || true
echo "✅ Schema applied"
echo ""

# ── STEP 2: Migrate data ───────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 2/4 — Migrating data in FK order..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

strip_and_import "data/users_dump.sql"              "users"
strip_and_import "data/materials_dump.sql"          "materials"
echo "  ⚠️  material_embeddings is 16MB — may take 30–90 seconds..."
strip_and_import "data/material_embeddings_dump.sql" "material_embeddings"
strip_and_import "data/chats_dump.sql"              "chats"
strip_and_import "data/messages_dump.sql"           "messages"
strip_and_import "data/query_log_dump.sql"          "query_log"

echo ""
echo "✅ All tables imported"
echo ""

# ── STEP 3: Reset sequences ────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 3/4 — Resetting sequences..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
psql "$NEON_URL" << 'SQL'
SELECT setval('materials_id_seq',  (SELECT COALESCE(MAX(id),1) FROM materials));
SELECT setval('users_id_seq',      (SELECT COALESCE(MAX(id),1) FROM users));
SELECT setval('chats_id_seq',      (SELECT COALESCE(MAX(id),1) FROM chats));
SELECT setval('messages_id_seq',   (SELECT COALESCE(MAX(id),1) FROM messages));
SELECT setval('query_log_id_seq',  (SELECT COALESCE(MAX(id),1) FROM query_log));
SQL
echo "✅ Sequences reset"
echo ""

# ── STEP 4: Verify ────────────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 4/4 — Final row counts on Neon:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
psql "$NEON_URL" << 'SQL'
SELECT 'materials'           AS tbl, COUNT(*) FROM materials
UNION ALL SELECT 'material_embeddings', COUNT(*) FROM material_embeddings
UNION ALL SELECT 'users',              COUNT(*) FROM users
UNION ALL SELECT 'chats',              COUNT(*) FROM chats
UNION ALL SELECT 'messages',           COUNT(*) FROM messages
UNION ALL SELECT 'query_log',          COUNT(*) FROM query_log
ORDER BY tbl;
SQL

echo ""
echo "🎉 Migration complete! All data is on Neon."
