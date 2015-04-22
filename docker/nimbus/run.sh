#!/bin/bash
NAME=nimbus-dev-nimbus
WORKDIR=/go/src/github.com/bearfrieze/nimbus
PORT=8080
PG=nimbus-dev-postgres
docker run --rm --name $NAME -v $HOME/go:/go -w $WORKDIR -p $PORT:$PORT -it --link $PG:pg $NAME