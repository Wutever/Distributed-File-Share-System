#!/usr/bin/env bash
USER=$1

for i in 1 2 3 4 5 6 7 8 9 10; do
    VM="fa18-cs425-g27-"
    if [ $i != 10 ]; then
        VM+="0"
    fi
    VM=$USER"@"$VM$i".cs.illinois.edu"
    echo $VM

    scp -r ~/ece428/mp2 $VM:
done
