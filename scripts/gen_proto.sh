#!/bin/bash

function check_installed() {
  if ! command -v $1 &> /dev/null
  then
    echo "$1 could not be found"
    exit
  fi
}

check_installed protoc

if [ -z "$2" ]
then
  echo "Usage: gen_proto.sh <path-to-perun-wallet-spec> <target-dir>"
  exit
fi

# Directory where perun-wallet-spec is located:
PROTO_DIR=$1
TARGET_DIR=$2

protoc --go_out=$TARGET_DIR \
  --go-grpc_out=$TARGET_DIR \
  --go_opt=Mperun-wallet.proto=proto/ \
  --go-grpc_opt=Mperun-wallet.proto=proto/ \
  --proto_path=$PROTO_DIR/src/proto \
  $PROTO_DIR/src/proto/perun-wallet.proto
