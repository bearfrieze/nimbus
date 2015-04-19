#!/bin/bash
docker run -it --name psql --rm --link pg:pg psql