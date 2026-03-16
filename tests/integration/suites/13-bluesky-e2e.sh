# Suite 13 — Bluesky End-to-End
#
# Full lifecycle test against real Bluesky using a pre-provisioned account:
#   Create connection → Create post → Publish → Monitor
#
# Requires env vars:
#   WERD_TEST_BLUESKY_IDENTIFIER    (e.g. "testuser.bsky.social")
#   WERD_TEST_BLUESKY_APP_PASSWORD  (app password for API adapter)
#
# Skips all tests if credentials are not provided.

suite "Bluesky End-to-End"

BLUESKY_IDENTIFIER="${WERD_TEST_BLUESKY_IDENTIFIER:-}"
BLUESKY_APP_PASSWORD="${WERD_TEST_BLUESKY_APP_PASSWORD:-}"

if [ -z "$BLUESKY_IDENTIFIER" ] || [ -z "$BLUESKY_APP_PASSWORD" ]; then
  skip "Bluesky E2E (WERD_TEST_BLUESKY_* env vars not set)"
  return
fi

login_setup || { fail "login_setup failed"; return; }

TEST_ID=$(generate_test_id)

# ── Step 1: Create Bluesky API connection ──
conn_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  "{\"platform\":\"bluesky\",\"method\":\"api\",\"credentials\":{\"identifier\":\"$BLUESKY_IDENTIFIER\",\"app_password\":\"$BLUESKY_APP_PASSWORD\"}}")
BSKY_CONN_ID=$(echo "$conn_resp" | jq -r '.id // empty')

if [ -n "$BSKY_CONN_ID" ]; then
  pass "Create Bluesky API connection (id=$BSKY_CONN_ID)"
else
  error_msg=$(echo "$conn_resp" | jq -r '.message // empty')
  fail "Create Bluesky API connection (error: $error_msg)"
  return
fi

# ── Step 2: Create draft post ──
post_content="Werd E2E test ${TEST_ID} — automated integration test, please ignore"

post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  "{\"content\":\"$post_content\",\"post_type\":\"text\",\"platforms\":[\"bluesky\"]}")
BSKY_POST_ID=$(echo "$post_resp" | jq -r '.id // empty')

if [ -n "$BSKY_POST_ID" ]; then
  pass "Create draft post for Bluesky (id=$BSKY_POST_ID)"
else
  fail "Create draft post (response: $post_resp)"
  return
fi

# ── Step 3: Publish to Bluesky ──
echo "  Publishing to Bluesky..."
publish_resp=$(curl -sf -X POST \
  -H "Authorization: Bearer $TEST_TOKEN" \
  -H "Content-Type: application/json" \
  "$CADDY_API/api/projects/$TEST_PROJECT_ID/posts/$BSKY_POST_ID/publish" 2>/dev/null)

bsky_result=$(echo "$publish_resp" | jq -r '.results[] | select(.platform == "bluesky")')
bsky_success=$(echo "$bsky_result" | jq -r '.success')
bsky_post_url=$(echo "$bsky_result" | jq -r '.url // empty')

if [ "$bsky_success" = "true" ]; then
  pass "Publish to Bluesky succeeded (url=$bsky_post_url)"
else
  bsky_error=$(echo "$bsky_result" | jq -r '.error // empty')
  fail "Publish to Bluesky failed (error: $bsky_error)"
  return
fi

# ── Step 4: Verify post status ──
get_post=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$BSKY_POST_ID")
assert_json_field "$get_post" ".status" "published" "Bluesky post status is published"

published_at=$(echo "$get_post" | jq -r '.published_at // empty')
if [ -n "$published_at" ] && [ "$published_at" != "null" ]; then
  pass "Bluesky post has published_at timestamp"
else
  fail "Bluesky post missing published_at"
fi

# ── Step 5: Create account monitor source ──
source_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/sources" \
  '{"type":"bluesky","config":{"mode":"account","poll_interval_secs":600},"enabled":true}')
BSKY_SOURCE_ID=$(echo "$source_resp" | jq -r '.id // empty')

if [ -n "$BSKY_SOURCE_ID" ]; then
  pass "Create Bluesky account monitor (id=$BSKY_SOURCE_ID)"
else
  fail "Create Bluesky account monitor (response: $source_resp)"
fi

# ── Step 6: Enable reply monitoring ──
monitor_resp=$(api_put "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$BSKY_POST_ID/monitor" \
  '{"enable":true}')
if echo "$monitor_resp" | jq -e '.message' >/dev/null 2>&1; then
  pass "Enable Bluesky reply monitoring"
else
  fail "Enable Bluesky reply monitoring (response: $monitor_resp)"
fi
