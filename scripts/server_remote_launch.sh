#!/bin/bash
nohup $HOME/ter-grpc serve \
    --port 1313 \
    --certificate $HOME/localhost.cert \
    --key $HOME/localhost.key \
    --compress \
    --workers-addresses $2 \
    > $HOME/server_logs.txt 2> $HOME/server_error_logs.txt &
echo $! > $1