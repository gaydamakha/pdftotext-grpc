#!/bin/bash
#In this script we suppose that "make" and "make certs" are already done

### DEPLOY PART BEGIN ###
user=gaydamakha
mapfile -t machines < ./machines.txt
# #TODO: check for availavility and take the first available
server=$user@${machines[0]}
scp $GOPATH/bin/ter-grpc ../certs/localhost.cert ../certs/localhost.key  ./launch.sh $server:~
# Launch server
ssh $server ./launch.sh save_pid.txt
### DEPLOY PART END ###

### METRICS PART BEGIN
address=$(ssh $server 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
$GOPATH/bin/ter-grpc pdftotext --root-certificate ../certs/localhost.cert --file ../fixtures/small.pdf --address $address:1313 --iters 5 --result-fn ./results.txt
#TODO: collect different metrics
#TODO: write them to the file

### METRICS PART END
#Be clean
ssh $server 'kill `cat save_pid.txt`'
