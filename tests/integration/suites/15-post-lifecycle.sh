# Suite 15 — Post Lifecycle & State Machine
#
# Tests post CRUD operations and the draft → published state machine.

suite "Post Lifecycle"

login_setup || { fail "login_setup failed"; return; }

# Create an HN API connection for publishing tests.
api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections" \
  '{"platform":"hn","method":"api","credentials":{}}' >/dev/null

# ── Create draft post ──
post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  '{"title":"Test Post","content":"Test content","post_type":"text","platforms":["hn"]}')
POST_ID=$(echo "$post_resp" | jq -r '.id // empty')
if [ -n "$POST_ID" ]; then
  pass "Create draft post (id=$POST_ID)"
else
  fail "Create draft post (response: $post_resp)"
fi
assert_json_field "$post_resp" ".status" "draft" "New post has status=draft"

# ── List posts ──
list_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts")
list_count=$(echo "$list_resp" | jq '.posts | length')
if [ "$list_count" -ge 1 ]; then
  pass "List posts returns at least 1"
else
  fail "List posts (count=$list_count)"
fi

# ── Get single post ──
get_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$POST_ID")
assert_json_field "$get_resp" ".title" "Test Post" "Get post returns correct title"

# ── Update draft ──
update_resp=$(api_put "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$POST_ID" \
  '{"title":"Updated Title","content":"Updated content","post_type":"text","platforms":["hn"]}')
assert_json_field "$update_resp" ".title" "Updated Title" "Update post changes title"

# ── Create and delete a draft ──
del_post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  '{"title":"To Delete","content":"Delete me","post_type":"text","platforms":["hn"]}')
DEL_POST_ID=$(echo "$del_post_resp" | jq -r '.id // empty')
del_status=$(api_delete_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$DEL_POST_ID")
assert_eq "200" "$del_status" "Delete draft post returns 200"

# ── Create post with no platforms → error ──
no_plat_status=$(api_post_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  '{"title":"No Platforms","content":"Test","post_type":"text","platforms":[]}')
assert_eq "400" "$no_plat_status" "Post with no platforms returns 400"

# ── Create post with no content → error ──
no_content_status=$(api_post_status "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  '{"post_type":"text","platforms":["hn"]}')
assert_eq "400" "$no_content_status" "Post with no content returns 400"

# ── Create link post ──
link_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  '{"title":"Link Post","url":"https://example.com","post_type":"link","platforms":["hn"]}')
assert_json_field "$link_resp" ".post_type" "link" "Link post has post_type=link"
assert_json_field "$link_resp" ".url" "https://example.com" "Link post has correct URL"
