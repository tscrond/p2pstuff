#!/bin/bash

if [[ "$1" == "build" ]]; then
    go build -o ./bin/p1/
    go build -o ./bin/p2/
    go build -o ./bin/p3/
    go build -o ./bin/p4/
    go build -o ./bin/bootstrap/
fi
