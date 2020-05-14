#!/bin/bash
# In this script we suppose that "make" and "make certs" are already done

DIRNAME="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SRC_DIR="$DIRNAME/../"
SERVER_FILE="$SRC_DIR/server.txt"
MACHINES_FILE="$SRC_DIR/machines.txt"
FIXTURES_DIR="$SRC_DIR/fixtures"
RESULTS_DIR="$SRC_DIR/results"
TXT_DIR="$SRC_DIR/txt"
NB_FILES=20
SERVER_PORT=5000
USER=gaydamakha
MAX_CHUNK_SIZE=2097152
ARR_NB_WORKERS=(1 2 3 4)
NB_ITERS=3
#Try to repeat a measure 3 times before abandon
MAX_TRIES=3
BIN=$GOPATH/bin/ter-grpc

if [[ ! -f "$MACHINES_FILE" ]]; then
    echo "File $MACHINES_FILE does not exists"
    exit 1
fi

mkdir -p $RESULTS_DIR
mkdir -p $TXT_DIR

#Launch different number of workers
for NB_WORKERS in "${ARR_NB_WORKERS[@]}"; do
    RESULTS_FN=$RESULTS_DIR/results-nbwrks-$NB_WORKERS.txt
    #The initial server chunk size value
    SERVER_CHUNK_SIZE=16
    while [ $SERVER_CHUNK_SIZE -le $MAX_CHUNK_SIZE ]; do
        #Deploy the server with this current chunk and number of workers values
        echo "Deploying the server with chunksize=$SERVER_CHUNK_SIZE and nb_workers=$NB_WORKERS"
        $DIRNAME/icps_deploy.sh -f $MACHINES_FILE -u $USER -p $SERVER_PORT -n $NB_WORKERS -c $SERVER_CHUNK_SIZE -b $BIN >/dev/null
        if [[ ! -f "$SERVER_FILE" ]]; then
            echo "File $SERVER_FILE does not exists"
            #Retry to deploy the same configuration
            continue
        fi
        #Fetch server IP
        SERVER_IP=$(cat $SERVER_FILE | cut -d ' ' -f2)
        #The initial client chunk size value
        CLIENT_CHUNK_SIZE=16
        while [[ $CLIENT_CHUNK_SIZE -le $MAX_CHUNK_SIZE ]]; do
            for FILENAME in $FIXTURES_DIR/*.pdf; do
                echo "Launching client with chunk_size:$CLIENT_CHUNK_SIZE file:$FILENAME"
                #Make a measure 3 times to calculate an average value
                ITER=0
                TIME=0
                NB_SUCCESS=0
                while [[ $ITER -lt $NB_ITERS ]]; do
                    FILENAME_BASE=$(basename -- "$FILENAME")
                    mkdir -p $TXT_DIR
                    CODE=1
                    TRY=1
                    ITER_TIME=0
                    #repeat until successfull or number of tries is achieved
                    until [[ $CODE -eq 0 || $TRY -gt $MAX_TRIES ]]; do
                        ITER_TIME=$($GOPATH/bin/ter-grpc pdftotext --bidirectional=true --compress=true --root-certificate $SRC_DIR/certs/localhost.cert \
                            --file $FILENAME --address $SERVER_IP:$SERVER_PORT --iters $NB_FILES --txt-dir $TXT_DIR)
                        CODE=$?
                        ((TRY++))
                    done
                    #Add the result if successfull
                    if [[ $CODE -eq 0 ]]; then
                        ((NB_SUCCESS++))
                        echo "Success $NB_SUCCESS: adding the time iter time $ITER_TIME to time $TIME"
                        TIME=$(awk "BEGIN {print $TIME+$ITER_TIME; exit}")
                    fi
                    ((ITER++))
                done
                #Calculate the average time
                if [[ $NB_SUCCESS -gt 0 ]]; then
                    echo "Success tries: $NB_SUCCESS. Divide time $TIME by it"
                    TIME=$(awk "BEGIN {print $TIME/$NB_SUCCESS; exit}")
                    echo "time: $TIME server_chunk_size: $SERVER_CHUNK_SIZE client_chunk_size: $CLIENT_CHUNK_SIZE"
                    echo "time: $TIME server_chunk_size: $SERVER_CHUNK_SIZE client_chunk_size: $CLIENT_CHUNK_SIZE" >>$RESULTS_FN
                fi
                # clean up downloaded files
                rm -f $TXT_DIR/*
                CLIENT_CHUNK_SIZE=$(($CLIENT_CHUNK_SIZE * 2))
            done
        done
        $DIRNAME/icps_stop.sh >/dev/null
        SERVER_CHUNK_SIZE=$(($SERVER_CHUNK_SIZE * 2))
    done
done
