#!/bin/bash
NAME=nimbus-dev-psql
PG=nimbus-dev-postgres
docker run -it --name $NAME --rm --link $PG:pg $NAME