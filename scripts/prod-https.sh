#!/usr/bin/env bash
# Public HTTPS in front of the docker stack (Let's Encrypt, DuckDNS or any hostname).
# Company login stays on https://$PUBLIC_HOST/c/<slug> so a wildcard cert is not required.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
# shellcheck source=prod-env.sh
source "$root/scripts/prod-env.sh"

env_file="$root/.env"
if [[ ! -f "$env_file" ]]; then
  echo "Нет .env — скопируйте .env.example и заполните блок Production HTTPS." >&2
  exit 1
fi

host=$(read_dotenv "$env_file" PUBLIC_HOST)
origins=$(read_dotenv "$env_file" API_ALLOWED_ORIGINS)
secure=$(read_dotenv "$env_file" SESSION_COOKIE_SECURE)
rpid=$(read_dotenv "$env_file" WEBAUTHN_RP_ID)

if [[ -z "$host" ]]; then
  echo "Задайте PUBLIC_HOST в .env, например orderfill.duckdns.org" >&2
  exit 1
fi
if [[ "$host" == *://* || "$host" == */* ]]; then
  echo "PUBLIC_HOST — только имя, без https:// и пути: $host" >&2
  exit 1
fi
if [[ "${secure}" != "true" && "${secure}" != "1" && "${secure}" != "yes" ]]; then
  echo "Поставьте SESSION_COOKIE_SECURE=true" >&2
  exit 1
fi
if [[ "$rpid" != "$host" ]]; then
  echo "WEBAUTHN_RP_ID должен совпадать с PUBLIC_HOST ($host), сейчас: ${rpid:-пусто}" >&2
  exit 1
fi
if [[ ",${origins}," != *",https://${host},"* && "$origins" != "https://${host}" ]]; then
  echo "API_ALLOWED_ORIGINS должен содержать https://${host}" >&2
  exit 1
fi

check_production_env "$env_file"
bash "$root/scripts/gen-internal-tls.sh"

echo "==> поднимаю production stack + https для ${host}"
docker compose --env-file "$env_file" \
  -f "$root/deploy/docker-compose.yml" \
  -f "$root/deploy/docker-compose.prod.yml" \
  -f "$root/deploy/docker-compose.https.yml" \
  up -d --build

cat <<EOF

HTTPS: https://${host}
Вход компании: https://${host}/c/<адрес-входа>
Passkey работает в Safari/Chrome на этом адресе.

На DuckDNS A-запись должна смотреть на белый IP сервера.
В файрволе откройте 80 и 443. Порт 3200 снаружи больше не нужен.
Postgres/Redis/MinIO с host не публикуются. Внутренний gRPC — mTLS.
EOF
