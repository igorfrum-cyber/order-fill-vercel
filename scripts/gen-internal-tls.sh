#!/usr/bin/env bash
# Internal CA + one leaf cert per Compose service (gRPC mTLS, Postgres SSL, MinIO TLS).
# Output is gitignored under deploy/secrets/grpc. CA key stays on the host and is
# never mounted into containers. FORCE=1 rotates the CA and every leaf.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
dir="$root/deploy/secrets/grpc"
ca_crt="$dir/ca.crt"
ca_key="$dir/ca.key"
names=(
  gateway-service
  identity-service
  twofa-service
  passkey-service
  job-service
  file-service
  document-api
  document-worker
  matching-service
  brand-service
  calculation-service
  audit-service
  postgres
  minio
)

if ! command -v openssl >/dev/null 2>&1; then
  echo "internal tls: openssl is required" >&2
  exit 1
fi

umask 077
mkdir -p "$dir"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Drop the previous shared-cert layout; those keys were mounted into every container.
rm -f "$dir/order-fill-grpc.crt" "$dir/order-fill-grpc.key" "$dir/order-fill-grpc-ca.crt"

need_ca=0
if [[ "${FORCE:-}" == "1" || ! -f "$ca_crt" || ! -f "$ca_key" ]]; then
  need_ca=1
fi

need_leaf=0
for name in "${names[@]}"; do
  if [[ ! -f "$dir/$name/tls.crt" || ! -f "$dir/$name/tls.key" ]]; then
    need_leaf=1
    break
  fi
done

if [[ "$need_ca" -eq 0 && "$need_leaf" -eq 0 && "${FORCE:-}" != "1" ]]; then
  echo "internal tls: reuse $dir"
  exit 0
fi

if [[ "$need_ca" -eq 1 ]]; then
  cat >"$tmp/ca.cnf" <<'EOF'
[req]
distinguished_name = dn
x509_extensions = v3_ca
prompt = no
[dn]
CN = order-fill-internal-ca
[v3_ca]
basicConstraints = critical,CA:TRUE
keyUsage = critical,keyCertSign,cRLSign
subjectKeyIdentifier = hash
EOF
  openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
    -keyout "$ca_key" -out "$ca_crt" -config "$tmp/ca.cnf" >/dev/null
  chmod 600 "$ca_key"
  chmod 644 "$ca_crt"
  need_leaf=1
fi

issue_leaf() {
  local name=$1
  local out="$dir/$name"
  mkdir -p "$out"
  cat >"$tmp/${name}.cnf" <<EOF
[req]
distinguished_name = dn
prompt = no
[dn]
CN = ${name}
[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth,clientAuth
subjectAltName = DNS:${name}
EOF
  openssl req -newkey rsa:2048 -nodes -keyout "$out/tls.key" -out "$tmp/${name}.csr" \
    -config "$tmp/${name}.cnf" >/dev/null
  openssl x509 -req -in "$tmp/${name}.csr" -CA "$ca_crt" -CAkey "$ca_key" \
    -CAcreateserial -out "$out/tls.crt" -days 825 \
    -extfile "$tmp/${name}.cnf" -extensions v3_req >/dev/null
  chmod 600 "$out/tls.key"
  chmod 644 "$out/tls.crt"
}

if [[ "$need_leaf" -eq 1 || "${FORCE:-}" == "1" ]]; then
  for name in "${names[@]}"; do
    issue_leaf "$name"
  done
fi

openssl verify -CAfile "$ca_crt" \
  "$dir/gateway-service/tls.crt" "$dir/identity-service/tls.crt" >/dev/null

if cmp -s "$dir/gateway-service/tls.key" "$dir/identity-service/tls.key"; then
  echo "internal tls: service keys must differ" >&2
  exit 1
fi

gw_san=$(openssl x509 -in "$dir/gateway-service/tls.crt" -noout -ext subjectAltName)
if [[ "$gw_san" != *DNS:gateway-service* || "$gw_san" == *identity-service* ]]; then
  echo "internal tls: gateway SAN must be only gateway-service" >&2
  exit 1
fi

echo "internal tls: wrote per-service certs in $dir"
