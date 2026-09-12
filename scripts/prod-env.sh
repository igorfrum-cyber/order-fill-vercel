#!/usr/bin/env bash
# Shared production .env checks. Sourced by prod-https.sh and deploy.sh.
# Do not print secret values.

read_dotenv() {
  local file=$1 key=$2
  local line
  line=$(grep -E "^${key}=" "$file" | tail -n 1 || true)
  printf '%s' "${line#*=}" | tr -d '"' | tr -d "'"
}

check_production_env() {
  local env_file=$1
  local app_env worker twofa mode db queue s3_key s3_secret s3_ssl pg redis

  app_env=$(read_dotenv "$env_file" APP_ENV)
  if [[ "$app_env" != "production" ]]; then
    echo "Для production поставьте APP_ENV=production" >&2
    return 1
  fi

  worker=$(read_dotenv "$env_file" WORKER_TOKEN)
  if [[ -z "$worker" || "$worker" == "local-dev-worker-token" || ${#worker} -lt 16 ]]; then
    echo "WORKER_TOKEN: свой секрет ≥16 байт, не local-dev-worker-token" >&2
    return 1
  fi

  twofa=$(read_dotenv "$env_file" TWOFA_MASTER_KEY)
  if [[ -z "$twofa" || "$twofa" == "local-dev-twofa-master-key" || ${#twofa} -lt 32 ]]; then
    echo "TWOFA_MASTER_KEY: свой секрет ≥32 байт, не local-dev-twofa-master-key" >&2
    return 1
  fi

  mode=$(read_dotenv "$env_file" GRPC_TLS_MODE)
  if [[ "$mode" != "mtls" ]]; then
    echo "GRPC_TLS_MODE=mtls обязателен в production (сертификаты создаст scripts/gen-internal-tls.sh)" >&2
    return 1
  fi

  db=$(read_dotenv "$env_file" DATABASE_URL)
  if [[ -z "$db" || "$db" == *":order_fill@"* ]]; then
    echo "DATABASE_URL: не используйте пароль order_fill" >&2
    return 1
  fi
  if [[ "$db" != *"sslmode=require"* && "$db" != *"sslmode=verify-ca"* && "$db" != *"sslmode=verify-full"* ]]; then
    echo "DATABASE_URL должен задавать sslmode=require (или verify-ca / verify-full)" >&2
    return 1
  fi

  queue=$(read_dotenv "$env_file" QUEUE_URL)
  if [[ "$queue" != redis://*:*@* ]]; then
    echo "QUEUE_URL должен включать пароль Redis, например redis://:<password>@redis:6379/0" >&2
    return 1
  fi

  s3_key=$(read_dotenv "$env_file" S3_ACCESS_KEY)
  s3_secret=$(read_dotenv "$env_file" S3_SECRET_KEY)
  if [[ -z "$s3_key" || -z "$s3_secret" || "$s3_key" == "minioadmin" || "$s3_secret" == "minioadmin" ]]; then
    echo "S3_ACCESS_KEY / S3_SECRET_KEY: не minioadmin" >&2
    return 1
  fi

  s3_ssl=$(read_dotenv "$env_file" FILE_S3_USE_SSL)
  if [[ "$s3_ssl" != "true" && "$s3_ssl" != "1" && "$s3_ssl" != "yes" ]]; then
    echo "FILE_S3_USE_SSL=true обязателен в production" >&2
    return 1
  fi

  pg=$(read_dotenv "$env_file" POSTGRES_PASSWORD)
  if [[ -z "$pg" || "$pg" == "order_fill" ]]; then
    echo "POSTGRES_PASSWORD: не order_fill" >&2
    return 1
  fi

  redis=$(read_dotenv "$env_file" REDIS_PASSWORD)
  if [[ -z "$redis" ]]; then
    echo "REDIS_PASSWORD обязателен в production" >&2
    return 1
  fi
}
