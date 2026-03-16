# Suite 10 — Platform Connections CRUD
#
# Tests creating, listing, updating, and deleting platform connections
# via the Werd API.

suite "Platform Connections CRUD"

login_setup || { fail "login_setup failed"; return; }

# ── Create HN API connection ──
conn_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  '{"platform":"hn","method":"api","credentials":{}}')
CONN_ID=$(echo "$conn_resp" | jq -r '.id // empty')
if [ -n "$CONN_ID" ]; then
  pass "Create HN API connection (id=$CONN_ID)"
else
  fail "Create HN API connection (response: $conn_resp)"
fi

# ── List connections ──
list_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections")
list_count=$(echo "$list_resp" | jq 'length')
if [ "$list_count" -ge 1 ]; then
  pass "List connections returns at least 1"
else
  fail "List connections (count=$list_count)"
fi

# ── Credentials not leaked ──
if echo "$list_resp" | grep -q '"credentials"'; then
  fail "Credentials should not appear in list response"
else
  pass "Credentials not leaked in list response"
fi

# ── Get single connection ──
get_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections/$CONN_ID")
assert_json_field "$get_resp" ".platform" "hn" "Get connection returns correct platform"
assert_json_field "$get_resp" ".method" "api" "Get connection returns correct method"

# ── Update connection ──
update_resp=$(api_put "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections/$CONN_ID" \
  '{"platform":"hn","method":"api","credentials":{},"enabled":false}')
assert_json_field "$update_resp" ".enabled" "false" "Update connection disables it"

# ── Delete connection ──
del_status=$(api_delete_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections/$CONN_ID")
assert_eq "200" "$del_status" "Delete connection returns 200"

# ── Verify deleted ──
get_status=$(curl -s -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $TEST_TOKEN" \
  "$CADDY_API/api/projects/$TEST_PROJECT_ID/connections/$CONN_ID" 2>/dev/null)
assert_eq "404" "$get_status" "Deleted connection returns 404"

# ── Invalid method ──
bad_method_status=$(api_post_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  '{"platform":"hn","method":"invalid","credentials":{}}')
assert_eq "400" "$bad_method_status" "Invalid method returns 400"

# ── Unsupported platform ──
bad_platform_status=$(api_post_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  '{"platform":"twitter","method":"api","credentials":{}}')
assert_eq "400" "$bad_platform_status" "Unsupported platform returns 400"
