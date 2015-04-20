#!/bin/bash
NAME=nimbus-dev-postgres
docker stop $NAME
docker rm $NAME
docker run --name $NAME -p 5432:5432 -d postgres