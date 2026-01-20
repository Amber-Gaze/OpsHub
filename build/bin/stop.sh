#!/bin/bash

pid=`ps -ef | grep '\./OpsHub' | grep -v grep | awk '{print $2}'`
if [ "$pid" == "" ]; then
    echo "OpsHub not running."
    exit 0
fi

echo "kill $pid"
kill $pid
if [ $? -ne 0 ]; then
    echo "kill $pid failed: $?, try 'kill -9 $pid'"
    kill -9 $pid
fi

exit $? 