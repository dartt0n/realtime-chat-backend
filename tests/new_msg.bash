#!/bin/bash

curl --request POST \
    --url http://localhost:8080/message \
    -H 'Authorization: '$TOKEN'' \
    --header 'User-Agent: insomnia/10.3.0'
