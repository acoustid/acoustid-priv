#!/usr/bin/env bash
if [ -t 1 ]
then
    exec docker-compose exec postgres psql -U acoustid acoustid_priv
else
    exec docker exec -i $(docker-compose ps -q postgres) psql -U acoustid acoustid_priv
fi
