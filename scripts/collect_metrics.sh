#!/bin/bash
#In this script we suppose that "make" and "make certs" are already done

DIRNAME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

### DEPLOY PART BEGIN ###
user=gaydamakha
mapfile -t machines < $DIRNAME/../machines.txt

server=$user@${machines[4]}
worker_server=$server
#TODO: launch multiple workers
scp $GOPATH/bin/ter-grpc $DIRNAME/../certs/localhost.cert $DIRNAME/../certs/localhost.key $DIRNAME/worker_remote_launch.sh $worker_server:~
# Launch worker
ssh $worker_server ./worker_remote_launch.sh save_worker_pid.txt
#END TODO

scp $DIRNAME/../certs/localhost.cert $DIRNAME/../certs/localhost.key $DIRNAME/server_remote_launch.sh $server:~
# Launch server
worker_ip=$(ssh $worker_server 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
ssh $server ./server_remote_launch.sh save_server_pid.txt $worker_ip:1314
### DEPLOY PART END ###

### METRICS PART BEGIN ###
address=$(ssh $server 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
$GOPATH/bin/ter-grpc pdftotext --bidirectional=true --root-certificate $DIRNAME/../certs/localhost.cert \
    --file  $DIRNAME/../fixtures/small.pdf --address $address:1313 --iters 5 \
    --result-fn $DIRNAME/../metrics/results.txt --txt-dir $DIRNAME/../metrics/
#TODO: collect different metrics

#END TODO

### METRICS PART END ###
#Be clean
ssh $server 'kill `cat save_server_pid.txt`'
#TODO: kill all the workers
ssh $worker_server 'kill `cat save_worker_pid.txt`'
#END TODO
