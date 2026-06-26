#! /bin/bash

cd web/default && bun run build && \
cd ../classic && bun run build && \
cd ../.. && \
docker build -t me2r036/new-api .
docker push me2r036/new-api
