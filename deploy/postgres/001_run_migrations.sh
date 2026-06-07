#!/bin/sh
set -eu

for migration in /docker-entrypoint-initdb.d/migrations/*.sql; do
  echo "applying migration ${migration}"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --file "$migration"
done
