# Suite 11 — Hacker News End-to-End
#
# Full lifecycle test against real HN:
#   Create account → Validate → Create post → Publish → Verify on HN → Monitor setup
#
# This is the primary E2E test — HN has no CAPTCHA so the full flow is automatable.

suite "Hacker News End-to-End"

login_setup || { fail "login_setup failed"; return; }

TEST_ID=$(generate_test_id)

# ── Step 1: Create HN account via browser automation ──
hn_username="wt${TEST_ID}"
# Truncate to 15 chars (HN username limit).
hn_username="${hn_username:0:15}"
hn_password="TestPass_${TEST_ID}!"

echo "  Creating HN account: $hn_username"

create_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections/create-account" \
  "{\"platform\":\"hn\",\"username\":\"$hn_username\",\"password\":\"$hn_password\"}")
HN_CONN_ID=$(echo "$create_resp" | jq -r '.connection.id // empty')

if [ -n "$HN_CONN_ID" ]; then
  pass "Create HN account and connection (id=$HN_CONN_ID)"
else
  error_msg=$(echo "$create_resp" | jq -r '.message // .error // empty')
  if echo "$error_msg" | grep -qi "rate-limited\|disabled"; then
    skip "HN account creation rate-limited — skipping remaining HN E2E tests"
    return
  fi
  fail "Create HN account (error: $error_msg)"
  # Cannot proceed without account — skip remaining tests.
  return
fi

# ── Step 2: Verify connection stored ──
conns_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections")
hn_conn=$(echo "$conns_resp" | jq -r ".[] | select(.id == \"$HN_CONN_ID\")")
if [ -n "$hn_conn" ]; then
  assert_json_field "$hn_conn" ".platform" "hn" "HN connection has correct platform"
  assert_json_field "$hn_conn" ".method" "browser" "HN connection has method=browser"
  assert_json_field "$hn_conn" ".enabled" "true" "HN connection is enabled"
else
  fail "HN connection not found in list"
fi

# ── Step 3: Create draft post ──
post_title="Werd E2E Test ${TEST_ID}"
post_body="Automated integration test from Werd. Please ignore."

post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  "{\"title\":\"$post_title\",\"content\":\"$post_body\",\"post_type\":\"text\",\"platforms\":[\"hn\"]}")
HN_POST_ID=$(echo "$post_resp" | jq -r '.id // empty')

if [ -n "$HN_POST_ID" ]; then
  pass "Create draft post (id=$HN_POST_ID)"
else
  fail "Create draft post (response: $post_resp)"
  return
fi
assert_json_field "$post_resp" ".status" "draft" "Post starts as draft"

# ── Step 4: Publish post ──
echo "  Publishing to HN (this may take 30-60s)..."
publish_resp=$(curl -sf -X POST \
  -H "Authorization: Bearer $TEST_TOKEN" \
  -H "Content-Type: application/json" \
  "$CADDY_API/api/projects/$TEST_PROJECT_ID/posts/$HN_POST_ID/publish" 2>/dev/null)
publish_status=$?

if [ $publish_status -eq 0 ]; then
  hn_result=$(echo "$publish_resp" | jq -r '.results[] | select(.platform == "hn")')
  hn_success=$(echo "$hn_result" | jq -r '.success')
  hn_post_url=$(echo "$hn_result" | jq -r '.url // empty')
  hn_platform_post_id=$(echo "$hn_result" | jq -r '.post_id // empty')

  if [ "$hn_success" = "true" ]; then
    pass "Publish to HN succeeded (url=$hn_post_url)"
  else
    hn_error=$(echo "$hn_result" | jq -r '.error // empty')
    fail "Publish to HN failed (error: $hn_error)"
    return
  fi
else
  fail "Publish request failed (curl exit=$publish_status)"
  return
fi

# ── Step 5: Verify post status ──
get_post=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$HN_POST_ID")
assert_json_field "$get_post" ".status" "published" "Post status is published"

# Check published_at is set.
published_at=$(echo "$get_post" | jq -r '.published_at // empty')
if [ -n "$published_at" ] && [ "$published_at" != "null" ]; then
  pass "Post has published_at timestamp"
else
  fail "Post missing published_at"
fi

# ── Step 6: Verify post exists on HN ──
if [ -n "$hn_post_url" ] && [ "$hn_post_url" != "null" ]; then
  sleep 3  # Brief delay for HN to serve the page.
  hn_page=$(curl -sf "$hn_post_url" 2>/dev/null || echo "")
  if echo "$hn_page" | grep -q "$post_title"; then
    pass "Post title found on HN page"
  else
    # HN might not render title exactly — check for partial match.
    if [ -n "$hn_page" ]; then
      fail "Post title not found on HN page (URL: $hn_post_url)"
    else
      skip "Could not fetch HN page (URL: $hn_post_url)"
    fi
  fi
else
  skip "No HN URL to verify"
fi

# ── Step 7: Enable reply monitoring ──
monitor_resp=$(api_put "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$HN_POST_ID/monitor" \
  '{"enable":true}')
if echo "$monitor_resp" | jq -e '.message' >/dev/null 2>&1; then
  pass "Enable reply monitoring"
else
  fail "Enable reply monitoring (response: $monitor_resp)"
fi

# ── Step 8: Create monitor source for the HN thread ──
if [ -n "$hn_platform_post_id" ] && [ "$hn_platform_post_id" != "null" ]; then
  source_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/sources" \
    "{\"type\":\"hn\",\"config\":{\"mode\":\"thread\",\"item_id\":$hn_platform_post_id},\"enabled\":true}")
  SOURCE_ID=$(echo "$source_resp" | jq -r '.id // empty')
  if [ -n "$SOURCE_ID" ]; then
    pass "Create HN thread monitor source (id=$SOURCE_ID)"
  else
    fail "Create HN thread monitor source (response: $source_resp)"
  fi
else
  skip "No HN item ID — cannot create thread monitor"
fi

# ── Step 9: Verify source persisted ──
if [ -n "$SOURCE_ID" ]; then
  sources_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/sources")
  source_found=$(echo "$sources_resp" | jq -r ".[] | select(.id == \"$SOURCE_ID\") | .id")
  if [ -n "$source_found" ]; then
    pass "Monitor source found in list"
  else
    fail "Monitor source not found in list"
  fi
fi

# ── Step 10: Verify published post cannot be edited ──
update_status=$(api_put_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$HN_POST_ID" \
  '{"title":"Should Fail","content":"x","post_type":"text","platforms":["hn"]}')
assert_eq "409" "$update_status" "Cannot edit published post (409)"

# ── Step 11: Verify published post cannot be deleted ──
delete_status=$(api_delete_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$HN_POST_ID")
assert_eq "409" "$delete_status" "Cannot delete published post (409)"
