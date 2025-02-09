#!/bin/bash

curl --request POST \
    --url http://localhost:8080/message \
    --header 'Authorization: Bearer '$TOKEN'' \
    --header 'Content-Type: application/json' \
    --data '{"content": "hello world!"}'
