#!/bin/bash

if [ $# != 1 ] ; then
echo "USAGE: $0 port"
exit 1;
fi

port=$1

pids=`netstat -anp | grep $port | grep tcp | grep LISTEN | awk '{print $7}' | awk -F "/" '{print $1}'`
for pid in $pids
do
    kill -9 $pid &
    echo "kill process $pid"
    sleep 2
done
echo "kill processes on port $port"