#!/bin/bash

usage() { echo "Usage: -p <port> -a <addresses of workers> -c <chunk size> -b <path to binary>"; exit 1; }

while getopts ":p:a:c:b:" o; do
    case "${o}" in
        p)
            PORT=${OPTARG}
            ;;
        a)
            WORKERS=${OPTARG}
            ;;
        c)
            CHUNK_SIZE=${OPTARG}
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

if [[ -z "${WORKERS}" ]]; then
    # Worker's adresses must be specified
    usage
fi

if [[ -z "${CHUNK_SIZE}" ]]; then
    # Chunk size must be specified
    usage
fi

if [[ -z "${BIN}" ]]; then
    usage
fi

SRC_DIR=$HOME/source-code/
SERVER_DIR=$HOME/server

mkdir -p $SERVER_DIR

nohup $BIN \
    --port $PORT \
    --certificate $SRC_DIR/certs/localhost.cert \
    --key $SRC_DIR/certs/localhost.key \
    --compress \
    --chunk-size $CHUNK_SIZE \
    --workers "${WORKERS}" > $SERVER_DIR/logs.txt 2> $SERVER_DIR/error_logs.txt &

echo $! > $SERVER_DIR/pid.txt
