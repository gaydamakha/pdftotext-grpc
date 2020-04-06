#!/bin/bash
#In this script we suppose that "make" and "make certs" are already done

DIRNAME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

### DEPLOY PART BEGIN ###
user=gaydamakha
mapfile -t machines < $DIRNAME/../machines.txt
#TODO: check for availavility and take the first available
server=$user@${machines[4]}
scp $GOPATH/bin/ter-grpc $DIRNAME/../certs/localhost.cert $DIRNAME/../certs/localhost.key $DIRNAME/remote_launch.sh $server:~
# Launch server
ssh $server ./remote_launch.sh save_pid.txt
### DEPLOY PART END ###

### METRICS PART BEGIN
address=$(ssh $server 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
$GOPATH/bin/ter-grpc pdftotext --bidirectional=true --root-certificate $DIRNAME/../certs/localhost.cert --file  $DIRNAME/../fixtures/small.pdf --address $address:1313 --iters 5 --result-fn $DIRNAME/../metrics/results.txt --txt-dir $DIRNAME/../metrics/
#TODO: collect different metrics

### METRICS PART END
#Be clean
ssh $server 'kill `cat save_pid.txt`'
