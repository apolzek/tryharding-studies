#!/usr/bin/env bash
# Seeds the system with a handful of users, catalogs, products, and orders
# so the UI has realistic data to show from the start.
#
# Expects the stack to be reachable at $BASE (default http://localhost:8080).
# Safe to run more than once — creates are naturally idempotent for demo data.

set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
API="$BASE/api"

banner() { printf '\n\033[1;35m==> %s\033[0m\n' "$*"; }
post()   { curl -sS -X POST "$1" -H 'content-type: application/json' --data "$2"; }

banner "register demo user"
post "$API/auth/register" '{"email":"demo@057.test","password":"demo12345"}' || true
TOKEN=$(post "$API/auth/login" '{"email":"demo@057.test","password":"demo12345"}' | \
  grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo "token: ${TOKEN:0:32}…"

AUTH=(-H "Authorization: Bearer $TOKEN" -H 'content-type: application/json')

banner "catalogs (MongoDB via catalog-service)"
for c in "Electronics" "Books" "Apparel"; do
  curl -sS "${AUTH[@]}" -X POST "$API/catalogs" --data "{\"name\":\"$c\"}" >/dev/null && echo "  · $c"
done

banner "products (Rust/gRPC via BFF)"
for i in 1 2 3 4 5; do
  curl -sS "${AUTH[@]}" -X POST "$API/products" --data @- <<EOF >/dev/null
{"sku":"DEMO-$i","name":"Demo product $i","price":$(( 10 + i * 7 )),"stock":100,"description":"seed","catalog_id":"default"}
EOF
  echo "  · DEMO-$i"
done

banner "customers (Go/Postgres, emits customer.created)"
for n in Alice Bob Carol Dave; do
  curl -sS "${AUTH[@]}" -X POST "$API/customers" --data "{\"name\":\"$n\",\"email\":\"$n@057.test\",\"document\":\"$RANDOM\"}" >/dev/null && echo "  · $n"
done

banner "grab a product id for the checkout demo"
PRODUCT_ID=$(curl -sS "${AUTH[@]}" "$API/products" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo "  · $PRODUCT_ID"

banner "run a checkout saga (creates order, decrements stock, charges payment)"
curl -sS "${AUTH[@]}" -X POST "$API/checkout" --data @- <<EOF | head -c 400
{"name":"Eve","email":"eve@057.test","document":"$RANDOM",
 "product_id":"$PRODUCT_ID","quantity":1,"plan":"premium","amount":29.9}
EOF
echo

banner "done — visit $BASE and http://localhost:16686 for traces"
