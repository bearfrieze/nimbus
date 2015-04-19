#!/bin/bash
export PGPORT=$PG_PORT_5432_TCP_PORT
export PGHOST=$PG_PORT_5432_TCP_ADDR
export PGUSER=postgres
export PGDATABASE=postgres
env
go run main.go