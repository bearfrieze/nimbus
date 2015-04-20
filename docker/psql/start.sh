#!/bin/bash
export PGHOST=$PG_PORT_5432_TCP_ADDR
export PGPORT=$PG_PORT_5432_TCP_PORT
export PGDATABASE=postgres
export PGUSER=postgres
psql