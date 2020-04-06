#!/bin/bash
nohup $HOME/ter-grpc serve --certificate $HOME/localhost.cert --key $HOME/localhost.key > $HOME/logs.txt 2> $HOME/error_logs.txt &
echo $! > $1