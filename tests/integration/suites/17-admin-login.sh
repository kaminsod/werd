# Suite 17 — Admin Login & Project Access
#
# Verifies that the admin user can log in with the configured credentials
# and that they have access to projects (including previously orphaned ones).

suite "Admin Login & Project Access"

login_setup

# ── Login succeeds with configured credentials ──

login_resp=$(curl -s -w "\n%{http_code}" -X POST "$CADDY_API/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_ADMIN_EMAIL\",\"password\":\"$TEST_ADMIN_PASSWORD\"}")

login_body=$(echo "$login_resp" | head -1)
login_status=$(echo "$login_resp" | tail -1)

assert_eq "200" "$login_status" "POST /auth/login returns 200"

login_token=$(echo "$login_body" | jq -r '.token // empty')
if [ -n "$login_token" ]; then
  pass "Login response contains JWT token"
else
  fail "Login response missing JWT token"
fi

login_email=$(echo "$login_body" | jq -r '.user.email // empty')
assert_eq "$TEST_ADMIN_EMAIL" "$login_email" "Login response contains correct email"

# ── /auth/me works with the token ──

me_resp=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $login_token" "$CADDY_API/api/auth/me")
me_body=$(echo "$me_resp" | head -1)
me_status=$(echo "$me_resp" | tail -1)

assert_eq "200" "$me_status" "GET /auth/me returns 200"

me_email=$(echo "$me_body" | jq -r '.email // empty')
assert_eq "$TEST_ADMIN_EMAIL" "$me_email" "/auth/me returns correct email"

# ── Admin can list projects ──

projects_resp=$(api_get "$login_token" "/projects")
project_count=$(echo "$projects_resp" | jq 'if type == "array" then length else 0 end' 2>/dev/null || echo "0")

if [ "$project_count" -ge 1 ]; then
  pass "Admin can list projects ($project_count found)"
else
  # In integration tests, login_setup creates a project — so at least 1 should exist.
  fail "Admin has no projects (expected >= 1, got $project_count)"
fi

# ── Admin can create a new project ──

new_slug="logintest_$(date +%s)"
create_resp=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Authorization: Bearer $login_token" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Login Test Project\",\"slug\":\"$new_slug\"}" \
  "$CADDY_API/api/projects")

create_body=$(echo "$create_resp" | head -1)
create_status=$(echo "$create_resp" | tail -1)

assert_eq "201" "$create_status" "POST /projects returns 201 (create project)"

created_slug=$(echo "$create_body" | jq -r '.slug // empty')
assert_eq "$new_slug" "$created_slug" "Created project has correct slug"

# ── Duplicate slug is rejected ──

dup_status=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
  -H "Authorization: Bearer $login_token" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Dup Project\",\"slug\":\"$new_slug\"}" \
  "$CADDY_API/api/projects")

assert_eq "409" "$dup_status" "Duplicate slug returns 409 Conflict"

# ── Wrong password returns 401 with correct error message ──

wrong_resp=$(curl -s -w "\n%{http_code}" -X POST "$CADDY_API/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_ADMIN_EMAIL\",\"password\":\"wrongpassword\"}")

wrong_body=$(echo "$wrong_resp" | head -1)
wrong_status=$(echo "$wrong_resp" | tail -1)

assert_eq "401" "$wrong_status" "Wrong password returns 401"

wrong_msg=$(echo "$wrong_body" | jq -r '.message // empty')
assert_eq "invalid email or password" "$wrong_msg" "401 response has correct error message"
