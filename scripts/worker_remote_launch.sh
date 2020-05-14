#!/bin/bash

usage() { echo "Usage: $0 -i <worker_id> -p <port> -b <path to binary>"; exit 1; }

while getopts ":i:p:b:" o; do
    case "${o}" in
        i)
            WORKER_ID=${OPTARG}
            ;;
        p)
            PORT=${OPTARG}
            ;;
        b)
            BIN=${OPTARG}
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [[ -z "${PORT}" ]]; then
    # Port must be specified
    usage
fi

if [[ -z "${WORKER_ID}" ]]; then
    # Worker id must be specified
    usage
fi

if [[ -z "${BIN}" ]]; then
    usage
fi

SRC_DIR=$HOME/source-code/
WORKER_DIR=$HOME/worker_$WORKER_ID
LOGS_DIR=$WORKER_DIR/logs

# Will create the WORKER_DIR too
mkdir -p ${LOGS_DIR}

nohup $BIN worker-serve \
    --port 800${WORKER_ID} \
    --certificate $SRC_DIR/certs/localhost.cert \
    --key $SRC_DIR/certs/localhost.key > $LOGS_DIR/logs.txt 2> $LOGS_DIR/error_logs.txt &

echo $! > $WORKER_DIR/pid.txt
