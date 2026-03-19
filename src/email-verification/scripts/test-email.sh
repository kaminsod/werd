#!/usr/bin/env bash
# Send a test email and verify it arrives via the Mailpit API.
#
# Delivery methods (tried in order):
#   1. SMTP (smtplib) — works when outbound port 25 is not blocked
#   2. Mailpit send API — fallback when SMTP is unreachable
#
# Usage: ./scripts/test-email.sh [host] [smtp_port] [api_port]
set -euo pipefail

HOST="${1:-localhost}"
SMTP_PORT="${2:-25}"
API_PORT="${3:-8025}"
RECIPIENT="test-$(date +%s)@datazo.net"
SUBJECT="werd-test-$(date +%s)"

echo "Sending test email to ${RECIPIENT} via ${HOST}..."

# Try SMTP first, fall back to Mailpit send API
if python3 -c "
import smtplib
from email.mime.text import MIMEText
msg = MIMEText('This is a test email from werd email-verification.')
msg['Subject'] = '${SUBJECT}'
msg['From'] = 'test@test.local'
msg['To'] = '${RECIPIENT}'
with smtplib.SMTP('${HOST}', ${SMTP_PORT}, timeout=10) as s:
    s.send_message(msg)
" 2>/dev/null; then
  echo "Sent via SMTP (${HOST}:${SMTP_PORT})"
else
  echo "SMTP (${HOST}:${SMTP_PORT}) unreachable, using Mailpit send API..."
  curl -sf -X POST "http://${HOST}:${API_PORT}/api/v1/send" \
    -H "Content-Type: application/json" \
    -d "{
      \"From\": {\"Email\": \"test@test.local\"},
      \"To\": [{\"Email\": \"${RECIPIENT}\"}],
      \"Subject\": \"${SUBJECT}\",
      \"Text\": \"This is a test email from werd email-verification.\"
    }" > /dev/null || {
    echo "FAIL: Could not send via SMTP or Mailpit API"
    exit 1
  }
  echo "Sent via Mailpit API (${HOST}:${API_PORT})"
fi

echo "Waiting for delivery..."
sleep 3

# Verify via Mailpit API
echo "Checking Mailpit API at ${HOST}:${API_PORT}..."
RESPONSE=$(curl -sf "http://${HOST}:${API_PORT}/api/v1/search?query=to:${RECIPIENT}" 2>&1) || {
  echo "FAIL: Could not reach Mailpit API at http://${HOST}:${API_PORT}"
  exit 1
}

COUNT=$(echo "${RESPONSE}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('messages_count', 0))" 2>/dev/null || echo "0")

if [ "${COUNT}" -gt 0 ]; then
  MSG_SUBJECT=$(echo "${RESPONSE}" | python3 -c "import sys,json; print(json.load(sys.stdin)['messages'][0]['Subject'])" 2>/dev/null || echo "unknown")
  MSG_ID=$(echo "${RESPONSE}" | python3 -c "import sys,json; print(json.load(sys.stdin)['messages'][0]['ID'])" 2>/dev/null || echo "unknown")
  echo "OK: Email received"
  echo "  Subject: ${MSG_SUBJECT}"
  echo "  ID:      ${MSG_ID}"
  exit 0
else
  echo "FAIL: Email not found in Mailpit"
  echo "  Searched for: to:${RECIPIENT}"
  exit 1
fi
