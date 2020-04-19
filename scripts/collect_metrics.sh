#!/bin/bash
#In this script we suppose that "make" and "make certs" are already done

DIRNAME="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

usage() { echo "Usage: $0 -a <server address>" 1>&2; exit 1; }

while getopts ":a:" o; do
    case "${o}" in
        a)
            SERVER_AD=${OPTARG}
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [[ -z "${SERVER_AD}" ]]; then
    # Server address must be specified
    usage
fi

$GOPATH/bin/ter-grpc pdftotext --bidirectional=true --root-certificate $DIRNAME/../certs/localhost.cert \
    --file  $DIRNAME/../fixtures/small.pdf --address $SERVER_AD --iters 5 \
    --result-fn $DIRNAME/../metrics/results.txt --txt-dir $DIRNAME/../metrics/
#TODO: collect different metrics

#END TODO