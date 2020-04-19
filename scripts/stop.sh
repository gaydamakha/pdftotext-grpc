#!/bin/bash

WORKERS_FILE="workers.txt"
SERVER_FILE="server.txt"

mapfile -t WORKERS < $WORKERS_FILE

#Stop the server first
ssh $(cat $SERVER_FILE) ./server_remote_stop.sh

#Stop the workers
while read WORKER_INFO
do
    WORKER_AD=$(echo $WORKER_INFO | head -n1 | cut -d " " -f1)
    WORKER_ID=$(echo $WORKER_INFO | head -n1 | cut -d " " -f2)
    ssh ${WORKER_AD} ./worker_remote_stop.sh ${WORKER_ID}
done < $WORKERS_FILE

rm $WORKERS_FILE
rm $SERVER_FILE