#!/bin/bash
NAME=pg
docker stop $NAME
docker rm $NAME
docker run --name $NAME -p 5432:5432 -d postgres