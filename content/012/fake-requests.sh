#!/bin/bash

URL="http://localhost:5000"
DURATION_MIN=120

# Função para gerar um valor de sleep randômico entre 0.1 e 2
function random_sleep {
    awk -v min=0.1 -v max=2 'BEGIN{srand(); print min+rand()*(max-min)}'
}

# Função para mandar requests para o endpoint /
function send_requests_root {
    for ((i=0; i<DURATION_MIN; i++)); do
        curl -s "$URL/" > /dev/null
        sleep $(random_sleep)
    done
}

# Função para mandar requests para o endpoint /increment
function send_requests_increment {
    end=$((SECONDS+10))
    while [ $SECONDS -lt $end ]; do
        curl -s "$URL/increment" > /dev/null
        sleep 0.6
    done
}

# Função para mandar requests para o endpoint /decrement
function send_requests_decrement {
    end=$((SECONDS+10))
    while [ $SECONDS -lt $end ]; do
        curl -s "$URL/decrement" > /dev/null
        sleep 0.8
    done
}

# Loop principal
while true; do
    echo "Sending requests to / for $DURATION_MIN seconds..."
    send_requests_root

    echo "Incrementing..."
    send_requests_increment

    # echo "Decrementing..."
    # send_requests_decrement

    wait
done
