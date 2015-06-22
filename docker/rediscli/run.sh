#!/bin/bash
REDIS=nimbus-dev-redis
docker run -it --link $REDIS:redis --rm redis sh -c 'exec redis-cli -h "$REDIS_PORT_6379_TCP_ADDR" -p "$REDIS_PORT_6379_TCP_PORT"'