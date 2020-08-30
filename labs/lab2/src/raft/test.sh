#!/bin/bash

for i in $(seq 30)
do
    go test -run 2A
    if [ $? -ne 0 ]; then
        echo "failed"
    fi
done
