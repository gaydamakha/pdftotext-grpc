#!/bin/bash

usage() { echo "Usage: $0 -i <worker_id> -p <port>"; exit 1; }

while getopts ":i:p:" o; do
    case "${o}" in
        i)
            WORKER_ID=${OPTARG}
            ;;
        p)
            PORT=${OPTARG}
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

WORKER_DIR=${HOME}/worker_${WORKER_ID}
LOGS_DIR=${WORKER_DIR}/logs

# Will create a WORKER_DIR too
mkdir -p ${LOGS_DIR}

nohup $HOME/ter-grpc worker-serve \
    --port 800${WORKER_ID} \
    --certificate $HOME/localhost.cert \
    --key $HOME/localhost.key > $LOGS_DIR/logs.txt 2> $LOGS_DIR/error_logs.txt &

echo $! > ${WORKER_DIR}/pid.txt
