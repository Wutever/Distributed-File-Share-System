#!/usr/bin/env bash
USER=$1
VM1=$2
VM2=$3
VM3=$4
VM4=$5

VM1=$USER"@fa18-cs425-g27-"$VM1".cs.illinois.edu"
VM2=$USER"@fa18-cs425-g27-"$VM2".cs.illinois.edu"
VM3=$USER"@fa18-cs425-g27-"$VM3".cs.illinois.edu"
VM4=$USER"@fa18-cs425-g27-"$VM4".cs.illinois.edu"

scp -r ~/ece428/mp2 $VM1:
scp -r ~/ece428/mp2 $VM2:
scp -r ~/ece428/mp2 $VM3:
scp -r ~/ece428/mp2 $VM4:
