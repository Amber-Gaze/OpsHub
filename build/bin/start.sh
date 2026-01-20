#!/bin/bash

pid=`ps -ef | grep '\./OpsHub' | grep -v grep | awk '{print $2}'`
if [ "$pid" != "" ]; then
    echo "another OpsHub is running now."
    exit 1
fi

GODEBUG=gctrace=1 nohup ./OpsHub >../log/stdout 2>../log/stderr &

sleep 2
pid=`ps -ef | grep '\./OpsHub' | grep -v grep | awk '{print $2}'`
if [ "$pid" == "" ]; then
    echo "start OpsHub failed:"
    tail -n 1 ../log/stderr
    exit 1
fi