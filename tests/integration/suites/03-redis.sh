# Suite 03 — Redis
#
# Validates AUTH enforcement, per-database read/write and isolation,
# and that redis.conf tuning (maxmemory, AOF) is loaded.

suite "Redis"

# Helpers: run redis-cli inside the redis container.
redis_exec() {
  compose_exec redis redis-cli -a "$REDIS_PASSWORD" "$@" 2>/dev/null
}

redis_exec_noauth() {
  compose_exec redis redis-cli "$@" 2>&1
}

# ── Unauthenticated connection rejected ──
result=$(redis_exec_noauth PING)
if echo "$result" | grep -qi "NOAUTH\|Authentication required"; then
  pass "Unauthenticated connection rejected (AUTH required)"
else
  fail "Unauthenticated connection not rejected (got: $result)"
fi

# ── Authenticated connection succeeds ──
result=$(redis_exec PING)
assert_eq "PONG" "$result" "Authenticated connection succeeds"

# ── Write/read on DB 0 (Werd API) ──
redis_exec -n 0 SET werd_integration_test "hello_db0" >/dev/null
result=$(redis_exec -n 0 GET werd_integration_test)
assert_eq "hello_db0" "$result" "Write/read on DB 0 (Werd API)"
redis_exec -n 0 DEL werd_integration_test >/dev/null

# ── Write/read on DB 2 (RSSHub) ──
redis_exec -n 2 SET rsshub_integration_test "hello_db2" >/dev/null
result=$(redis_exec -n 2 GET rsshub_integration_test)
assert_eq "hello_db2" "$result" "Write/read on DB 2 (RSSHub)"
redis_exec -n 2 DEL rsshub_integration_test >/dev/null

# ── DB isolation: key in DB 0 not visible in DB 2 ──
redis_exec -n 0 SET isolation_test "only_in_db0" >/dev/null
result=$(redis_exec -n 2 GET isolation_test)
if [ -z "$result" ] || [ "$result" = "" ]; then
  pass "DB isolation: key in DB 0 not visible in DB 2"
else
  fail "DB isolation broken: key from DB 0 visible in DB 2 (got: $result)"
fi
redis_exec -n 0 DEL isolation_test >/dev/null

# ── maxmemory configured ──
result=$(redis_exec CONFIG GET maxmemory | tail -1)
# 256 MB = 268435456 bytes
assert_eq "268435456" "$result" "redis.conf: maxmemory = 256MB (268435456 bytes)"

# ── Eviction policy ──
result=$(redis_exec CONFIG GET maxmemory-policy | tail -1)
assert_eq "allkeys-lru" "$result" "redis.conf: maxmemory-policy = allkeys-lru"

# ── AOF persistence enabled ──
result=$(redis_exec CONFIG GET appendonly | tail -1)
assert_eq "yes" "$result" "redis.conf: appendonly = yes (AOF enabled)"
