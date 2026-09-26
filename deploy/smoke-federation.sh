#!/usr/bin/env bash
# Starts two Amethyst servers from the production image and checks that each
# has its own identity and its own database, and that each can reach the other
# at its canonical origin. Run from anywhere: `make smoke`.
set -euo pipefail
cd "$(dirname "$0")"

compose() { docker compose -f compose.yaml --profile federation "$@"; }
fail() { echo "FAIL: $*" >&2; exit 1; }

# On failure, print container logs so CI output explains what went wrong.
trap 'status=$?; if [ "$status" -ne 0 ]; then compose logs --no-color --tail 50 server-a server-b migrate-a migrate-b federation-dbs; fi' EXIT

compose up -d --build

wait_ready() {
  local url=$1
  for _ in $(seq 1 60); do
    if curl -fsS "$url/api/readyz" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  fail "$url did not become ready within 60s"
}

# check_server ORIGIN DATABASE: the server at ORIGIN reports ORIGIN, and its
# database's only local server row is ORIGIN.
check_server() {
  local origin=$1 database=$2 reported stored
  wait_ready "$origin"

  reported=$(curl -fsS "$origin/api/server" | jq -r .canonical_origin)
  [ "$reported" = "$origin" ] || fail "$origin reports canonical origin '$reported'"

  stored=$(compose exec -T postgres psql -U amethyst -d "$database" -tAc \
    "SELECT string_agg(canonical_origin, ',') FROM servers WHERE is_local")
  [ "$stored" = "$origin" ] || fail "$database records local server '$stored', want '$origin'"

  echo "ok  $origin reports itself and owns $database"
}

# check_reachable ORIGIN: the canonical origin resolves and answers from
# inside the servers' network, as a peer server would reach it. Uses wget, not
# curl: curl (like browsers) hard-codes *.localhost to its own loopback
# address, whereas Go and wget ask DNS, which resolves the Compose alias.
check_reachable() {
  local to=$1 reported
  reported=$(docker run --rm --network amethyst_default alpine:3.22 wget -qO- "$to/api/server" | jq -r .canonical_origin)
  [ "$reported" = "$to" ] || fail "$to is not reachable at its origin from inside the network (got '$reported')"
  echo "ok  $to is reachable from the servers' network"
}

check_server http://a.localhost:8081 amethyst_a
check_server http://b.localhost:8082 amethyst_b
check_reachable http://a.localhost:8081
check_reachable http://b.localhost:8082

echo "Two independent servers are running."
