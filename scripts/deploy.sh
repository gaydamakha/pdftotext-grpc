#!/bin/bash
# In this script we suppose that "make" and "make certs" are already done
# 
# The last machine in the given file will be the server,
# and the others will be the workers, if -n is not specified

usage() { echo "Usage: $0 -f <filename> -p <server port> [-u <username=$USER>] [-n <number of workers>]" 1>&2; exit 1; }

while getopts ":f:u:n:p:" o; do
    case "${o}" in
        f)
            MACHINES_FILE=${OPTARG}
            ;;
        u)
            USER_NAME=${OPTARG}
            ;;
        n)
            NB_WORKERS=${OPTARG}
            ;;
        p)
            SERVER_PORT=${OPTARG}
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [[ -z "${SERVER_PORT}" ]]; then
    # Server port must be specified
    usage
fi

if [[ -z "${MACHINES_FILE}" ]]; then
    # Filename must be specified
    usage
else
    if ! [[ -f "$MACHINES_FILE" ]]; then
        echo "$MACHINES_FILE is not a valid filename"
        usage
    fi
fi

# By default, number of workers is a number of machines - 1
DEFAULT_NB_WORKERS=$(( $(wc -l < $MACHINES_FILE)-1 ))

if [[ -z "${NB_WORKERS}" ]]; then
    # Default if not specified
    NB_WORKERS=$DEFAULT_NB_WORKERS
else
    if [[ ${NB_WORKERS} -gt ${DEFAULT_NB_WORKERS} ]]; then
        echo "Specified number of workers is greater then number of available machines."
        echo "Using $DEFAULT_NB_WORKERS as a number of workers."
        NB_WORKERS=$DEFAULT_NB_WORKERS
    fi
fi

if [[ -z "${USER_NAME}" ]]; then
    # Using current username by default
    USER_NAME=$USER
fi

DIRNAME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
mapfile -t MACHINES < $MACHINES_FILE

# Deploy common files
# NB: we know that given machines have shared NFS, so one scp is sufficient
scp $GOPATH/bin/ter-grpc \
    $DIRNAME/../certs/localhost.cert \
    $DIRNAME/../certs/localhost.key \
    $DIRNAME/worker_remote_launch.sh \
    $DIRNAME/worker_remote_stop.sh \
    $DIRNAME/server_remote_launch.sh \
    $DIRNAME/server_remote_stop.sh \
    ${MACHINES[0]}:~

if ! [[ $? -eq 0 ]]; then
    echo "File copy failed."
    exit 1
fi

WORKERS_IPS=""
WORKERS_FILE="workers.txt"
SERVER_FILE="server.txt"
# Deploy workers first
i=0
while [[ $i -lt $NB_WORKERS ]];
do
    WORKER_AD=${MACHINES[i]}
    WORKER_IP=$(ssh $WORKER_AD 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
    echo "Deploying the worker in address $WORKER_IP:800$i ($WORKER_AD)..."
    ssh $WORKER_AD ./worker_remote_launch.sh -i $i -p 800$i
    if [[ $? -ne 0 ]]; then
        echo "Error while deploying the worker $WORKER_AD"
    else
        #Add worker address to the list of used workers
        WORKERS_IPS="${WORKERS_IPS} ${WORKER_IP}:800$i"
        echo "$WORKER_AD $i" >> $WORKERS_FILE
        ((i++))
    fi
done

echo "Launched workers: $WORKERS_IPS"

#Deploy the server
SERVER_AD=${MACHINES[i]}
SERVER_IP=$(ssh $SERVER_AD 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
echo "Deploying the server in address $SERVER_IP:$SERVER_PORT ($SERVER_AD)..."
ssh $SERVER_AD ./server_remote_launch.sh -p $SERVER_PORT -a "\"${WORKERS_IPS}\""

if [[ $? -ne 0 ]]; then
    echo "Error while deploying the server"
    exit 1
else
    echo $SERVER_AD >> $SERVER_FILE
    echo "Success!"
fi