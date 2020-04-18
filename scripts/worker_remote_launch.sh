#!/bin/bash
nohup $HOME/ter-grpc worker-serve \
    --port 1314 \
    --certificate $HOME/localhost.cert \
    --key $HOME/localhost.key \
    > $HOME/worker_logs.txt 2> $HOME/worker_error_logs.txt &
echo $! > $1