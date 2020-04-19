#!/bin/bash

usage() { echo "Usage: -p <port> -a <addresses of workers>"; exit 1; }

while getopts ":p:a:" o; do
    case "${o}" in
        p)
            PORT=${OPTARG}
            ;;
        a)
            WORKERS=${OPTARG}
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

SERVER_DIR=$HOME/server

mkdir -p $SERVER_DIR

nohup $HOME/ter-grpc serve \
    --port $PORT \
    --certificate $HOME/localhost.cert \
    --key $HOME/localhost.key \
    --compress \
    --workers "${WORKERS}" > $SERVER_DIR/logs.txt 2> $SERVER_DIR/error_logs.txt &

echo $! > $SERVER_DIR/pid.txt
