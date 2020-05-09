#!/bin/bash
# In this script we suppose that "make" and "make certs" are already done

DIRNAME="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SERVER_FILE="$DIRNAME/../server.txt"
MACHINES_FILE="$DIRNAME/../machines.txt"
FIXTURES_DIR="$DIRNAME/../fixtures"
RESULTS_DIR="$DIRNAME/../results"
FILE_NUMBERS=20
SERVER_PORT=5000
USER=gaydamakha
MAX_CHUNK_SIZE=2097152
ARR_NB_WORKERS=(1 2 3 4)
#Try to repeat a metric 3 times before abandon
MAX_TRIES=3

if [[ ! -f "$MACHINES_FILE" ]]; then
    echo "File $MACHINES_FILE does not exists"
    exit 1
fi

mkdir -p $RESULTS_DIR

#Launch different number of workers
for NB_WORKERS in "${ARR_NB_WORKERS[@]}"; do
    #The initial server chunk size value
    SERVER_CHUNK_SIZE=16
    while [ $SERVER_CHUNK_SIZE -le $MAX_CHUNK_SIZE ]; do
        #Deploy the server with this current chunk and number of workers values
        echo "Deploying the server with chunksize=$SERVER_CHUNK_SIZE and nb_workers=$NB_WORKERS"
        $DIRNAME/deploy.sh -f $MACHINES_FILE -u $USER -p $SERVER_PORT -n $NB_WORKERS -c $SERVER_CHUNK_SIZE >/dev/null
        #Fetch server IP
        if [[ ! -f "$SERVER_FILE" ]]; then
            echo "File $SERVER_FILE does not exists"
            continue
        fi
        SERVER_AD=$(cat $SERVER_FILE)
        SERVER_IP=""
        until [[ -n "$SERVER_IP" ]]; do
            SERVER_IP=$(ssh $SERVER_AD 'echo $SSH_CONNECTION' | cut -d ' ' -f3)
            sleep 1
        done
        #The initial client chunk size value
        CLIENT_CHUNK_SIZE=16
        while [[ $CLIENT_CHUNK_SIZE -le $MAX_CHUNK_SIZE ]]; do
            for FILENAME in $FIXTURES_DIR/*.pdf; do
                #Try to make it 10 times to calculate an average value
                for iter in {1..1}; do
                    FILENAME_BASE=$(basename -- "$FILENAME")
                    echo "nb_workers: $NB_WORKERS server_chunk_size: $SERVER_CHUNK_SIZE client_chunk_size: $CLIENT_CHUNK_SIZE filename $FILENAME_BASE iteration: $iter"
                    RESULT_FN=$RESULTS_DIR/results-$iter-$SERVER_CHUNK_SIZE-$NB_WORKERS-$CLIENT_CHUNK_SIZE-$FILENAME_BASE.txt
                    TXT_DIR=$DIRNAME/../txt/$SERVER_CHUNK_SIZE/$NB_WORKERS/$CLIENT_CHUNK_SIZE/"${FILENAME_BASE%.*}"/$iter/
                    mkdir -p $TXT_DIR
                    CODE=1
                    TRY=1
                    #repeat until successfull or number of tries is limited
                    until [[ $CODE -eq 0 || $TRY -gt $MAX_TRIES ]]; do
                        $GOPATH/bin/ter-grpc pdftotext --bidirectional=true --root-certificate $DIRNAME/../certs/localhost.cert \
                            --file $FILENAME --address $SERVER_IP:$SERVER_PORT --iters $FILE_NUMBERS \
                            --result-fn $RESULT_FN --txt-dir $TXT_DIR
                        CODE=$?
                        ((TRY++))
                    done
                    
                    echo "nb_workers: $NB_WORKERS server_chunk_size: $SERVER_CHUNK_SIZE client_chunk_size: $CLIENT_CHUNK_SIZE filename $FILENAME_BASE iteration: $iter" >> $RESULT_FN
                done
                #TODO: calculate an average
                CLIENT_CHUNK_SIZE=$(($CLIENT_CHUNK_SIZE * 2))
            done
        done
        $DIRNAME/stop.sh >/dev/null
        SERVER_CHUNK_SIZE=$(($SERVER_CHUNK_SIZE * 2))
    done
done
