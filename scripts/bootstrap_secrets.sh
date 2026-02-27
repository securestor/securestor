#!/usr/bin/env sh
set -eu

SECRETS_DIR="${1:-./secrets}"
mkdir -p "$SECRETS_DIR"
umask 077

write_secret() {
  file="$1"
  generator="$2"
  if [ ! -f "$SECRETS_DIR/$file" ]; then
    sh -c "$generator" > "$SECRETS_DIR/$file"
    chmod 600 "$SECRETS_DIR/$file"
    echo "Created $SECRETS_DIR/$file"
  else
    echo "Exists  $SECRETS_DIR/$file (skipped)"
  fi
}

write_secret "postgres_password" "openssl rand -base64 32"
write_secret "postgres_replication_password" "openssl rand -base64 32"
write_secret "redis_password" "openssl rand -base64 32"
write_secret "jwt_secret" "openssl rand -base64 64"
write_secret "keycloak_db_password" "openssl rand -base64 32"

echo "\nSecret bootstrap completed."
echo "Set KEYCLOAK_ADMIN_PASSWORD and KEYCLOAK_DB_PASSWORD in .env (or shell env) before running docker compose."
