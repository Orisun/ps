#!/bin/bash

EXE_FILE=ps
GROUP=""
ROLE=""
PORT=0
MANAGER=""
PC=0

while [ -n "$1" ]
do
  case "$1" in
    -g)
        GROUP=$2
        shift
        ;;
    -r)
        ROLE=$2
        shift
        ;;
    -p)
        PORT=$2
        shift
        ;;
    -m)
        MANAGER=$2
        shift
        ;;
    -c)
        PC=$2
        shift
        ;;
    *)
        echo "$1 is not an option"
        ;;
  esac
  shift
done

if [ -f ./$EXE_FILE ];then
rm -rf $EXE_FILE
fi
go build -o $EXE_FILE

pids=`netstat -anp | grep $PORT | grep tcp | grep LISTEN | awk '{print $7}' | awk -F "/" '{print $1}'`
for pid in $pids
do
    kill -s TERM $pid &
    echo "kill process $pid"
    sleep 2
done
echo "kill processes on port $PORT"

if [ $ROLE == "manager" ];then
nohup ./$EXE_FILE --role $ROLE --port $PORT --group $GROUP --parameter_count $PC > $EXE_FILE"_"$ROLE.out &
elif [ $ROLE == "server" ];then
nohup ./$EXE_FILE --role $ROLE --port $PORT --manager $MANAGER > $EXE_FILE"_"$ROLE.out &
fi