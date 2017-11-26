#!/usr/bin/env bash
exec docker exec -i $(docker-compose ps -q postgres) psql -U acoustid acoustid
