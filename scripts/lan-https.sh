#!/usr/bin/env bash
# Local HTTPS in front of the docker stack so iPhone Face ID can be tested.
# Needs the same Wi-Fi, then a one-time trust of the local certificate on the phone.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

certs="$root/deploy/certs"
ca_root="$certs/ca"
mkdir -p "$certs" "$ca_root"

lan_ip=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || true)
if [[ -z "${lan_ip:-}" ]]; then
  echo "Не вижу IP Wi-Fi. Подключите Mac к той же сети, что и телефон." >&2
  exit 1
fi

local_host=$(scutil --get LocalHostName 2>/dev/null | tr '[:upper:]' '[:lower:]' | tr -d ' ')
sslip_host="${lan_ip}.sslip.io"
nip_host="${lan_ip}.nip.io"
bonjour_host=""
if [[ -n "${local_host:-}" ]]; then
  bonjour_host="${local_host}.local"
fi

if ! command -v mkcert >/dev/null 2>&1; then
  if ! command -v brew >/dev/null 2>&1; then
    echo "Нужен mkcert. Установите Homebrew и повторите make lan-https." >&2
    exit 1
  fi
  echo "==> ставлю mkcert"
  brew install mkcert
fi

export CAROOT="$ca_root"
echo "==> выпускаю сертификат для ${sslip_host} ${nip_host} ${bonjour_host}"
names=("$sslip_host" "$nip_host")
if [[ -n "$bonjour_host" ]]; then
  names+=("$bonjour_host")
fi
mkcert -cert-file "$certs/lan.pem" -key-file "$certs/lan-key.pem" "${names[@]}"
openssl x509 -in "$ca_root/rootCA.pem" -outform der -out "$certs/local-ca.cer"

https_hosts=("https://${sslip_host}" "https://${nip_host}")
caddy_sites=("https://${sslip_host}" "https://${nip_host}")
if [[ -n "$bonjour_host" ]]; then
  https_hosts+=("https://${bonjour_host}")
  caddy_sites+=("https://${bonjour_host}")
fi

{
  echo "{"
  echo "  auto_https off"
  echo "  admin off"
  echo "}"
  echo
  printf '%s' "${caddy_sites[0]}"
  i=1
  while [[ $i -lt ${#caddy_sites[@]} ]]; do
    printf ', %s' "${caddy_sites[$i]}"
    i=$((i + 1))
  done
  echo " {"
  echo "  tls /certs/lan.pem /certs/lan-key.pem"
  echo "  reverse_proxy frontend:80 {"
  echo "    header_up Host {host}"
  echo "    header_up X-Forwarded-Proto https"
  echo "  }"
  echo "}"
} > "$root/deploy/Caddyfile.lan"

env_file="$root/.env"
if [[ ! -f "$env_file" ]]; then
  cp "$root/.env.example" "$env_file"
fi

current=$(grep -E '^API_ALLOWED_ORIGINS=' "$env_file" | head -n 1 | cut -d= -f2- || true)
if [[ -z "$current" ]]; then
  current="http://127.0.0.1:3200,http://localhost:3200,http://${lan_ip}:3200"
fi
merged="$current"
for origin in "http://${lan_ip}:3200" "${https_hosts[@]}"; do
  case ",$merged," in
    *",$origin,"*) ;;
    *) merged="${merged},${origin}" ;;
  esac
done

python3 - "$env_file" "$merged" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
merged = sys.argv[2]
text = path.read_text()
line = f"API_ALLOWED_ORIGINS={merged}\n"
if "API_ALLOWED_ORIGINS=" in text:
    lines = []
    replaced = False
    for row in text.splitlines(True):
        if row.startswith("API_ALLOWED_ORIGINS=") and not replaced:
            lines.append(line)
            replaced = True
        else:
            lines.append(row)
    path.write_text("".join(lines))
else:
    path.write_text(text.rstrip() + "\n" + line)
PY

echo "==> поднимаю https на порту 443"
docker compose --env-file "$env_file" \
  -f "$root/deploy/docker-compose.yml" \
  -f "$root/deploy/docker-compose.lan-https.yml" \
  up -d --build frontend api-service document-service caddy

cat <<EOF

HTTPS готов. Face ID на iPhone заработает только после доверия сертификату.

1. На iPhone в Safari откройте:
   http://${lan_ip}:3200/local-ca.cer
   Установите профиль. Пароль телефона — если спросит.

2. Настройки → Основные → Об этом устройстве → Доверие сертификатов.
   Включите полное доверие для «mkcert …».

3. Откройте в Safari (не Chrome):
   https://${sslip_host}

Запасной адрес, если первый не откроется:
   https://${nip_host}
EOF
if [[ -n "$bonjour_host" ]]; then
  echo "   https://${bonjour_host}"
fi
echo
echo "На Mac по-прежнему: http://127.0.0.1:3200"
echo "Ключ, добавленный на 127.0.0.1, на https-адресе не подойдёт — Face ID нужно добавить уже там."
echo
