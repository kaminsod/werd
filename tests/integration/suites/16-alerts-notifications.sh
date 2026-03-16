# Suite 16 — Alerts & Notifications
#
# Tests alert ingestion, listing, status updates, keywords, and rules.

suite "Alerts & Notifications"

login_setup || { fail "login_setup failed"; return; }

# Read internal API key from .env.test
INTERNAL_KEY=$(grep '^WERD_INTERNAL_API_KEY=' "$TEST_ENV" | cut -d= -f2)

# ── Webhook ingestion with API key ──
ingest_resp=$(curl -sf -X POST "$CADDY_API/api/webhooks/ingest" \
  -H "Content-Type: application/json" \
  -H "X-Internal-Key: $INTERNAL_KEY" \
  -d "{\"project_id\":\"$TEST_PROJECT_ID\",\"title\":\"Test Alert\",\"content\":\"Alert content\",\"url\":\"https://example.com\",\"source_type\":\"web\",\"severity\":\"low\"}" 2>/dev/null)
if echo "$ingest_resp" | jq -e '.id' >/dev/null 2>&1; then
  pass "Webhook ingestion creates alert"
  ALERT_ID=$(echo "$ingest_resp" | jq -r '.id')
else
  fail "Webhook ingestion (response: $ingest_resp)"
  ALERT_ID=""
fi

# ── Webhook without API key ──
no_key_status=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$CADDY_API/api/webhooks/ingest" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"test","title":"No Auth"}' 2>/dev/null)
assert_eq "401" "$no_key_status" "Webhook without API key returns 401"

# ── List alerts ──
alerts_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/alerts")
alert_count=$(echo "$alerts_resp" | jq '.alerts | length' 2>/dev/null || echo "0")
if [ "$alert_count" -ge 1 ]; then
  pass "List alerts returns at least 1"
else
  fail "List alerts (count=$alert_count)"
fi

# ── Update alert status ──
if [ -n "$ALERT_ID" ]; then
  update_status=$(api_put_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/alerts/$ALERT_ID" \
    '{"status":"acknowledged"}')
  assert_eq "200" "$update_status" "Update alert status returns 200"
else
  skip "Update alert status (no alert ID)"
fi

# ── Create keyword ──
kw_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/keywords" \
  '{"keyword":"integration-test","match_type":"exact"}')
KW_ID=$(echo "$kw_resp" | jq -r '.id // empty')
if [ -n "$KW_ID" ]; then
  pass "Create keyword (id=$KW_ID)"
else
  fail "Create keyword (response: $kw_resp)"
fi

# ── List keywords ──
kw_list=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/keywords")
kw_count=$(echo "$kw_list" | jq 'length' 2>/dev/null || echo "0")
if [ "$kw_count" -ge 1 ]; then
  pass "List keywords returns at least 1"
else
  fail "List keywords (count=$kw_count)"
fi

# ── Delete keyword ──
if [ -n "$KW_ID" ]; then
  del_kw_status=$(api_delete_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/keywords/$KW_ID")
  assert_eq "200" "$del_kw_status" "Delete keyword returns 200"
fi

# ── Create notification rule ──
rule_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/rules" \
  '{"name":"Test Rule","channel":"dashboard","severity_filter":"low","enabled":true}')
RULE_ID=$(echo "$rule_resp" | jq -r '.id // empty')
if [ -n "$RULE_ID" ]; then
  pass "Create notification rule (id=$RULE_ID)"
else
  fail "Create notification rule (response: $rule_resp)"
fi

# ── List rules ──
rules_list=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/rules")
rules_count=$(echo "$rules_list" | jq 'length' 2>/dev/null || echo "0")
if [ "$rules_count" -ge 1 ]; then
  pass "List rules returns at least 1"
else
  fail "List rules (count=$rules_count)"
fi

# ── Delete rule ──
if [ -n "$RULE_ID" ]; then
  del_rule_status=$(api_delete_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/rules/$RULE_ID")
  assert_eq "200" "$del_rule_status" "Delete notification rule returns 200"
fi
