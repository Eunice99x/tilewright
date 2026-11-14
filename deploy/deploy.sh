#!/bin/sh
# Deploy Tilewright to the Contabo VPS. Usage: deploy/deploy.sh (repo root). Needs /opt/tilewright/deploy/.env on the server.
set -eu
HOST=root@185.174.253.240
DIR=/opt/tilewright
SITE=https://tilewright.younesouterbah.site
ssh $HOST "mkdir -p $DIR/src $DIR/deploy"
rsync -az --delete --exclude node_modules --exclude '.next*' --exclude .git --exclude '.env*' --exclude deploy/.env ./ $HOST:$DIR/src/
scp -q deploy/docker-compose.yml $HOST:$DIR/deploy/docker-compose.yml
ssh $HOST "set -e; cd $DIR/src && \
  docker build -q -t tilewright-api:latest api && \
  docker build -q -f Dockerfile.web --build-arg SITE_URL=$SITE -t tilewright-web:latest . && \
  cd $DIR/deploy && docker compose up -d --no-build && sleep 4 && \
  for f in $DIR/src/api/migrations/*.sql; do docker exec -i tw-db psql -q -U tw -d tw < \$f || true; done && \
  docker image prune -f >/dev/null && docker compose ps --format '{{.Name}} {{.Status}}'"
