# Suite 08 — Persistence
#
# Validates that data written to PostgreSQL and Redis survives a container
# restart, proving that volumes and AOF are working correctly.

suite "Persistence"

# ── PostgreSQL: data survives restart ──

# Write test data.
compose_exec postgres psql -U werd -d werd -c \
  "CREATE TABLE IF NOT EXISTS _integration_test (id serial PRIMARY KEY, val text);
   INSERT INTO _integration_test (val) VALUES ('persist_check');" >/dev/null 2>&1

# Restart the postgres container.
compose_cmd restart postgres >/dev/null 2>&1

# Wait for postgres to become healthy again.
pg_elapsed=0
while [ $pg_elapsed -lt 60 ]; do
  if compose_exec postgres pg_isready -U werd >/dev/null 2>&1; then
    break
  fi
  sleep 2
  pg_elapsed=$((pg_elapsed + 2))
done

# Read back.
result=$(compose_exec postgres psql -U werd -d werd -tAc \
  "SELECT val FROM _integration_test WHERE val='persist_check' LIMIT 1;" 2>&1)
assert_eq "persist_check" "$result" "PostgreSQL: data survives container restart"

# Cleanup.
compose_exec postgres psql -U werd -d werd -c \
  "DROP TABLE IF EXISTS _integration_test;" >/dev/null 2>&1

# ── Redis: AOF data survives restart ──

# Write test data.
compose_exec redis redis-cli -a "$REDIS_PASSWORD" -n 0 \
  SET persistence_test "aof_check" >/dev/null 2>&1

# Force AOF rewrite to ensure data is flushed to disk.
compose_exec redis redis-cli -a "$REDIS_PASSWORD" BGREWRITEAOF >/dev/null 2>&1
sleep 2

# Restart the redis container.
compose_cmd restart redis >/dev/null 2>&1

# Wait for redis to become healthy again.
redis_elapsed=0
while [ $redis_elapsed -lt 30 ]; do
  if compose_exec redis redis-cli -a "$REDIS_PASSWORD" PING 2>/dev/null | grep -q PONG; then
    break
  fi
  sleep 2
  redis_elapsed=$((redis_elapsed + 2))
done

# Read back.
result=$(compose_exec redis redis-cli -a "$REDIS_PASSWORD" -n 0 \
  GET persistence_test 2>/dev/null)
assert_eq "aof_check" "$result" "Redis: AOF data survives container restart"

# Cleanup.
compose_exec redis redis-cli -a "$REDIS_PASSWORD" -n 0 \
  DEL persistence_test >/dev/null 2>&1
