#!/bin/sh
# postgres-exporter entrypoint wrapper — monta o DATA_SOURCE_NAME a partir de
# docker secrets (NUNCA hardcoded em config/env fixa). A imagem oficial do
# postgres-exporter nao suporta *_FILE nativo, por isso lemos /run/secrets/*.
# Stream W-OBS.
set -eu

PG_USER="$(tr -d '\r\n' < /run/secrets/pg_user)"
PG_PASS="$(tr -d '\r\n' < /run/secrets/pg_pass)"

# PG_HOST / PG_DB / PG_SSLMODE vem do environment do servico (nao-secretos).
: "${PG_HOST:?PG_HOST is required}"
: "${PG_DB:?PG_DB is required}"
: "${PG_SSLMODE:=disable}"

export DATA_SOURCE_NAME="postgresql://${PG_USER}:${PG_PASS}@${PG_HOST}/${PG_DB}?sslmode=${PG_SSLMODE}"

# Limpa as vars para nao vazar no ambiente do processo exportado.
unset PG_USER PG_PASS

exec /bin/postgres_exporter
