#!/usr/bin/env bash
# E2E tests for SIWE + Message Template Admin APIs
# Prerequisites: kubectl port-forward svc/web3-api 18080:8080 -n web3 --context web3-lab
#
# Usage: ./scripts/test-siwe-e2e.sh

set -euo pipefail

API_URL="${API_URL:-http://localhost:18080}"
PASS=0
FAIL=0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then
    echo -e "  ${GREEN}✓${NC} $label"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $label (expected: $expected, got: $actual)"
    FAIL=$((FAIL+1))
  fi
}

assert_contains() {
  local label="$1" expected="$2" actual="$3"
  if [[ "$actual" == *"$expected"* ]]; then
    echo -e "  ${GREEN}✓${NC} $label"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $label (expected to contain: '$expected')"
    FAIL=$((FAIL+1))
  fi
}

assert_not_empty() {
  local label="$1" actual="$2"
  if [[ -n "$actual" ]]; then
    echo -e "  ${GREEN}✓${NC} $label"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}✗${NC} $label (was empty)"
    FAIL=$((FAIL+1))
  fi
}

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 1. Admin Message Templates: LIST ═══${NC}"
RESPONSE=$(curl -m 10 -sf -H "Cookie: ory_hydra_session_dev=$API_KEY" "$API_URL/api/v1/admin/message-templates")
TEMPLATE_COUNT=$(echo "$RESPONSE" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('templates',[])))")
assert_eq "GET /admin/message-templates returns templates" "1" "$([[ $TEMPLATE_COUNT -ge 1 ]] && echo 1 || echo 0)"

EXISTING_ID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['templates'][0]['id'])")
assert_not_empty "First template has ID" "$EXISTING_ID"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 2. Admin Message Templates: GET ═══${NC}"
RESPONSE=$(curl -m 10 -sf -H "Cookie: ory_hydra_session_dev=$API_KEY" "$API_URL/api/v1/admin/message-templates/$EXISTING_ID")
TMPL_NAME=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['name'])")
assert_eq "GET by ID returns correct template" "default" "$TMPL_NAME"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 3. Admin Message Templates: CREATE ═══${NC}"
CREATE_BODY='{"name":"e2e-test","protocol":"siwe","statement":"E2E test statement","domain":"test.example.com","uri":"https://test.example.com","chain_id":42,"version":"1","nonce_ttl_secs":120}'
RESPONSE=$(curl -m 10 -sf -H "Cookie: ory_hydra_session_dev=$API_KEY" -X POST "$API_URL/api/v1/admin/message-templates" \
  -H "Content-Type: application/json" \
  -d "$CREATE_BODY")
NEW_ID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
assert_not_empty "POST creates template and returns ID" "$NEW_ID"
NEW_DOMAIN=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['domain'])")
assert_eq "Created template has correct domain" "test.example.com" "$NEW_DOMAIN"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 4. Admin Message Templates: UPDATE ═══${NC}"
UPDATE_BODY='{"name":"e2e-test-updated","statement":"Updated statement","domain":"test.example.com","uri":"https://test.example.com","chain_id":42,"version":"1","nonce_ttl_secs":180}'
RESPONSE=$(curl -m 10 -sf -H "Cookie: ory_hydra_session_dev=$API_KEY" -X PUT "$API_URL/api/v1/admin/message-templates/$NEW_ID" \
  -H "Content-Type: application/json" \
  -d "$UPDATE_BODY")
UPDATED_NAME=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['name'])")
assert_eq "PUT updates template name" "e2e-test-updated" "$UPDATED_NAME"
UPDATED_TTL=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['nonce_ttl_secs'])")
assert_eq "PUT updates nonce_ttl_secs" "180" "$UPDATED_TTL"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 5. Admin Message Templates: DELETE ═══${NC}"
HTTP_CODE=$(curl -m 10 -s -H "Cookie: ory_hydra_session_dev=$API_KEY" -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/admin/message-templates/$NEW_ID")
assert_eq "DELETE returns 204" "204" "$HTTP_CODE"
# Verify it's gone
HTTP_CODE=$(curl -m 10 -s -H "Cookie: ory_hydra_session_dev=$API_KEY" -o /dev/null -w "%{http_code}" "$API_URL/api/v1/admin/message-templates/$NEW_ID")
assert_eq "GET deleted template returns 404 or 500" "1" "$([[ "$HTTP_CODE" == "404" || "$HTTP_CODE" == "500" ]] && echo 1 || echo 0)"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 6. SIWE Nonce: SIWE Protocol ═══${NC}"
RESPONSE=$(curl -m 10 -sf "$API_URL/api/v1/siwe/nonce?address=0xAbCdEf1234567890AbCdEf1234567890AbCdEf12&protocol=siwe")
NONCE=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['nonce'])")
MSG=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['message'])")
PROTO=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['protocol'])")
assert_not_empty "SIWE nonce is generated" "$NONCE"
assert_eq "Protocol is siwe" "siwe" "$PROTO"
assert_contains "Message contains domain" "app.web3-local-dev.com wants you to sign in" "$MSG"
assert_contains "Message contains address" "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12" "$MSG"
assert_contains "Message contains nonce" "Nonce: $NONCE" "$MSG"
assert_contains "Message contains URI" "URI: https://app.web3-local-dev.com" "$MSG"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 7. SIWE Nonce: EIP-712 Protocol ═══${NC}"
RESPONSE=$(curl -m 10 -sf "$API_URL/api/v1/siwe/nonce?address=0xAbCdEf1234567890AbCdEf1234567890AbCdEf12&protocol=eip712")
PROTO=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['protocol'])")
MSG=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin)['message'])")
assert_eq "Protocol is eip712" "eip712" "$PROTO"
# EIP-712 message should be valid JSON with typed data
PRIMARY_TYPE=$(echo "$MSG" | python3 -c "import sys,json; print(json.load(sys.stdin)['primaryType'])")
assert_eq "EIP-712 primaryType is AuthMessage" "AuthMessage" "$PRIMARY_TYPE"
EIP_DOMAIN=$(echo "$MSG" | python3 -c "import sys,json; print(json.load(sys.stdin)['domain']['name'])")
assert_eq "EIP-712 domain name" "app.web3-local-dev.com" "$EIP_DOMAIN"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 8. SIWE Nonce: Invalid Protocol ═══${NC}"
HTTP_CODE=$(curl -m 10 -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/siwe/nonce?address=0x1234&protocol=invalid")
assert_eq "Invalid protocol returns 400 or 500" "1" "$([[ "$HTTP_CODE" == "400" || "$HTTP_CODE" == "500" ]] && echo 1 || echo 0)"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══ 9. SIWE Nonce: Missing Address ═══${NC}"
HTTP_CODE=$(curl -m 10 -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/siwe/nonce?protocol=siwe")
assert_eq "Missing address returns 400" "400" "$HTTP_CODE"

# ──────────────────────────────────────────────────
echo -e "\n${YELLOW}═══════════════════════════════════════${NC}"
TOTAL=$((PASS + FAIL))
echo -e "Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}, ${TOTAL} total"
if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
echo -e "${GREEN}All tests passed ✅${NC}"
