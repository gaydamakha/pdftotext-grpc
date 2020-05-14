#!/bin/bash

DIRNAME="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SRC_DIR="$DIRNAME/../"
SERVER_FILE="$SRC_DIR/server.txt"
WORKERS_FILE="$SRC_DIR/workers.txt"

if [[ ! -f "$SERVER_FILE" ]]; then
    echo "File $SERVER_FILE does not exists"
    exit 1
fi

#Stop the server first
CODE=1
until [[ CODE -eq 0 ]]; do
    SERVER_AD=$(cat $SERVER_FILE | cut -d ' ' -f1)
    echo "Stopping the server $SERVER_AD..."
    ssh $SERVER_AD ./server_remote_stop.sh
    CODE=$?
    sleep 1
done

rm -f $SERVER_FILE

if [[ ! -f "$WORKERS_FILE" ]]; then
    echo "File $WORKERS_FILE does not exists"
    exit 1
fi
mapfile -t WORKERS <$WORKERS_FILE

#Stop the workers
while IFS= read -r WORKER_INFO <&3 || [[ -n "$WORKER_INFO" ]]; do
    WORKER_AD=$(echo $WORKER_INFO | head -n1 | cut -d " " -f1)
    WORKER_ID=$(echo $WORKER_INFO | head -n1 | cut -d " " -f2)
    echo "Stopping the worker $WORKER_AD..."
    CODE=1
    until [[ CODE -eq 0 ]]; do
        ssh ${WORKER_AD} ./worker_remote_stop.sh ${WORKER_ID}
        CODE=$?
        sleep 1
    done
done 3<$WORKERS_FILE

rm -f $WORKERS_FILE
