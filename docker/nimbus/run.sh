#!/bin/bash
WORKDIR=/go/src/github.com/bearfrieze/nimbus
PORT=80
PG=nimbus-dev-postgres
docker run --rm --name nimbus-dev -v $HOME/go:/go -w $WORKDIR -p $PORT:$PORT -it --link $PG:pg nimbus-dev