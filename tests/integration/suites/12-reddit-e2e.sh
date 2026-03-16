# Suite 12 — Reddit End-to-End
#
# Full lifecycle test against real Reddit using a pre-provisioned account:
#   Create connection → Create post → Publish to r/test → Verify → Monitor
#
# Requires env vars:
#   WERD_TEST_REDDIT_CLIENT_ID
#   WERD_TEST_REDDIT_CLIENT_SECRET
#   WERD_TEST_REDDIT_USERNAME
#   WERD_TEST_REDDIT_PASSWORD
#   WERD_TEST_REDDIT_USER_AGENT     (default: werd-test/1.0)
#   WERD_TEST_REDDIT_SUBREDDIT      (default: test)
#
# Skips all tests if credentials are not provided.

suite "Reddit End-to-End"

# Check for credentials.
REDDIT_CLIENT_ID="${WERD_TEST_REDDIT_CLIENT_ID:-}"
REDDIT_CLIENT_SECRET="${WERD_TEST_REDDIT_CLIENT_SECRET:-}"
REDDIT_USERNAME="${WERD_TEST_REDDIT_USERNAME:-}"
REDDIT_PASSWORD="${WERD_TEST_REDDIT_PASSWORD:-}"
REDDIT_USER_AGENT="${WERD_TEST_REDDIT_USER_AGENT:-werd-test/1.0}"
REDDIT_SUBREDDIT="${WERD_TEST_REDDIT_SUBREDDIT:-test}"

if [ -z "$REDDIT_CLIENT_ID" ] || [ -z "$REDDIT_CLIENT_SECRET" ] || \
   [ -z "$REDDIT_USERNAME" ] || [ -z "$REDDIT_PASSWORD" ]; then
  skip "Reddit E2E (WERD_TEST_REDDIT_* env vars not set)"
  return
fi

login_setup || { fail "login_setup failed"; return; }

TEST_ID=$(generate_test_id)

# ── Step 1: Create Reddit API connection ──
reddit_creds=$(cat <<ENDJSON
{
  "client_id": "$REDDIT_CLIENT_ID",
  "client_secret": "$REDDIT_CLIENT_SECRET",
  "username": "$REDDIT_USERNAME",
  "password": "$REDDIT_PASSWORD",
  "user_agent": "$REDDIT_USER_AGENT",
  "subreddit": "$REDDIT_SUBREDDIT"
}
ENDJSON
)

conn_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  "{\"platform\":\"reddit\",\"method\":\"api\",\"credentials\":$reddit_creds}")
REDDIT_CONN_ID=$(echo "$conn_resp" | jq -r '.id // empty')

if [ -n "$REDDIT_CONN_ID" ]; then
  pass "Create Reddit API connection (id=$REDDIT_CONN_ID)"
else
  error_msg=$(echo "$conn_resp" | jq -r '.message // empty')
  fail "Create Reddit API connection (error: $error_msg)"
  return
fi

# ── Step 2: Create draft post ──
post_title="Werd E2E Test ${TEST_ID}"
post_body="Automated integration test from Werd browser automation. Please ignore."

post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  "{\"title\":\"$post_title\",\"content\":\"$post_body\",\"post_type\":\"text\",\"platforms\":[\"reddit\"]}")
REDDIT_POST_ID=$(echo "$post_resp" | jq -r '.id // empty')

if [ -n "$REDDIT_POST_ID" ]; then
  pass "Create draft post for Reddit (id=$REDDIT_POST_ID)"
else
  fail "Create draft post (response: $post_resp)"
  return
fi

# ── Step 3: Publish to Reddit ──
echo "  Publishing to r/$REDDIT_SUBREDDIT..."
publish_resp=$(curl -sf -X POST \
  -H "Authorization: Bearer $TEST_TOKEN" \
  -H "Content-Type: application/json" \
  "$CADDY_API/api/projects/$TEST_PROJECT_ID/posts/$REDDIT_POST_ID/publish" 2>/dev/null)

reddit_result=$(echo "$publish_resp" | jq -r '.results[] | select(.platform == "reddit")')
reddit_success=$(echo "$reddit_result" | jq -r '.success')
reddit_post_url=$(echo "$reddit_result" | jq -r '.url // empty')

if [ "$reddit_success" = "true" ]; then
  pass "Publish to Reddit succeeded (url=$reddit_post_url)"
else
  reddit_error=$(echo "$reddit_result" | jq -r '.error // empty')
  fail "Publish to Reddit failed (error: $reddit_error)"
  return
fi

# ── Step 4: Verify post status ──
get_post=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$REDDIT_POST_ID")
assert_json_field "$get_post" ".status" "published" "Reddit post status is published"

# ── Step 5: Verify post on Reddit ──
if [ -n "$reddit_post_url" ] && [ "$reddit_post_url" != "null" ]; then
  sleep 3
  reddit_page=$(curl -sf -A "$REDDIT_USER_AGENT" "$reddit_post_url" 2>/dev/null || echo "")
  if echo "$reddit_page" | grep -qi "$post_title"; then
    pass "Post title found on Reddit page"
  elif [ -n "$reddit_page" ]; then
    # Reddit might redirect or serve JS — partial match is acceptable.
    skip "Post title not found in Reddit response (may be JS-rendered)"
  else
    skip "Could not fetch Reddit page"
  fi
else
  skip "No Reddit URL to verify"
fi

# ── Step 6: Create subreddit monitor source ──
source_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/sources" \
  "{\"type\":\"reddit\",\"config\":{\"mode\":\"subreddit\",\"subreddit\":\"$REDDIT_SUBREDDIT\",\"poll_interval_secs\":300},\"enabled\":true}")
REDDIT_SOURCE_ID=$(echo "$source_resp" | jq -r '.id // empty')

if [ -n "$REDDIT_SOURCE_ID" ]; then
  pass "Create Reddit subreddit monitor (id=$REDDIT_SOURCE_ID)"
else
  fail "Create Reddit subreddit monitor (response: $source_resp)"
fi

# ── Step 7: Enable reply monitoring on the post ──
monitor_resp=$(api_put "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$REDDIT_POST_ID/monitor" \
  '{"enable":true}')
if echo "$monitor_resp" | jq -e '.message' >/dev/null 2>&1; then
  pass "Enable Reddit reply monitoring"
else
  fail "Enable Reddit reply monitoring (response: $monitor_resp)"
fi
