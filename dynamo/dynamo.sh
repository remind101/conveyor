#!/bin/bash

BASE=dynamodb-local

download() {
  if [ ! -f "dynamo.tar.gz" -o ! -s "dynamo.tar.gz" ]; then
    curl -L https://s3-us-west-2.amazonaws.com/dynamodb-local/dynamodb_local_latest.tar.gz > dynamo.tar.gz
  fi
}

install() {
  mkdir -p dynamodb-local
  tar -xzvf dynamo.tar.gz -C $BASE
}

run() {
  java -Djava.library.path=./$BASE/DynamoDBLocal_lib -jar ./$BASE/DynamoDBLocal.jar $@
}

case "$1" in
  install)
    if [ ! -d $BASE ]; then
      download
      install
    fi
    ;;
  run)
    shift
    run $@
esac
