# Suite 14 — Multi-Platform Publishing
#
# Tests cross-platform post publishing — a single post targeting all
# available platforms. Runs only if at least 2 platform connections exist.

suite "Multi-Platform Publishing"

login_setup || { fail "login_setup failed"; return; }

# ── Detect which platforms have connections ──
conns_resp=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/connections")
available_platforms=$(echo "$conns_resp" | jq -r '.[].platform' 2>/dev/null | sort -u)
platform_count=$(echo "$available_platforms" | grep -c . || true)

if [ "$platform_count" -lt 2 ]; then
  skip "Multi-platform test (need at least 2 platform connections, have $platform_count)"
  return
fi

echo "  Available platforms: $(echo $available_platforms | tr '\n' ' ')"

# Build platforms JSON array from available.
platforms_json=$(echo "$available_platforms" | jq -R -s 'split("\n") | map(select(. != ""))')

TEST_ID=$(generate_test_id)

# ── Create multi-platform post ──
post_resp=$(api_post "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts" \
  "{\"title\":\"Multi-Platform Test ${TEST_ID}\",\"content\":\"Cross-platform integration test. Ignore.\",\"post_type\":\"text\",\"platforms\":$platforms_json}")
MULTI_POST_ID=$(echo "$post_resp" | jq -r '.id // empty')

if [ -n "$MULTI_POST_ID" ]; then
  pass "Create multi-platform post (id=$MULTI_POST_ID, platforms=$platforms_json)"
else
  fail "Create multi-platform post (response: $post_resp)"
  return
fi

# ── Publish to all platforms ──
echo "  Publishing to all platforms (this may take a while)..."
publish_resp=$(curl -s -X POST \
  -H "Authorization: Bearer $TEST_TOKEN" \
  -H "Content-Type: application/json" \
  "$CADDY_API/api/projects/$TEST_PROJECT_ID/posts/$MULTI_POST_ID/publish" 2>/dev/null)
result_count=$(echo "$publish_resp" | jq '.results | length' 2>/dev/null || echo "0")
if [ "$result_count" -ge 2 ]; then
  pass "Publish returned results for $result_count platforms"
else
  fail "Expected results for at least 2 platforms (got $result_count)"
fi

# ── Check per-platform results ──
echo "$publish_resp" | jq -r '.results[] | "\(.platform): success=\(.success) url=\(.url // "N/A") error=\(.error // "none")"' 2>/dev/null | while read -r line; do
  platform_name=$(echo "$line" | cut -d: -f1)
  if echo "$line" | grep -q "success=true"; then
    pass "Platform $platform_name published successfully"
  else
    error_part=$(echo "$line" | grep -o 'error=.*')
    fail "Platform $platform_name failed ($error_part)"
  fi
done

# ── Check final status ──
get_post=$(api_get "$TEST_TOKEN" "/projects/$TEST_PROJECT_ID/posts/$MULTI_POST_ID")
final_status=$(echo "$get_post" | jq -r '.status // empty')

succeeded=$(echo "$publish_resp" | jq '[.results[] | select(.success == true)] | length' 2>/dev/null || echo "0")
failed=$(echo "$publish_resp" | jq '[.results[] | select(.success == false)] | length' 2>/dev/null || echo "0")

if [ "$failed" -eq 0 ]; then
  assert_eq "published" "$final_status" "All platforms succeeded — status=published"
else
  # Mixed results: status could be "failed" with a 207 response.
  echo "  Mixed results: $succeeded succeeded, $failed failed — status=$final_status"
  if [ "$final_status" = "published" ] || [ "$final_status" = "failed" ]; then
    pass "Post has valid terminal status ($final_status)"
  else
    fail "Post has unexpected status ($final_status)"
  fi
fi
