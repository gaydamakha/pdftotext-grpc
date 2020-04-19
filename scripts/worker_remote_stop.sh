#!/bin/bash

usage() { echo "Usage: $0 <worker_id>"; exit 1; }

if [[ $# -ne 1 ]]; then
    usage
fi

if ! [[ $1 =~ ^[0-9]+$ ]]; then
    usage
fi

WORKER_ID=$1
WORKER_DIR=$HOME/worker_$WORKER_ID

kill $(cat $WORKER_DIR/pid.txt)
