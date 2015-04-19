#!/bin/bash
WORKDIR=/go/src/github.com/bearfrieze/nimbus
PORT=8080
docker run --rm --name nimbus-dev -v $HOME/go:/go -w $WORKDIR --publish $PORT:$PORT -it --link pg:pg nimbus-dev