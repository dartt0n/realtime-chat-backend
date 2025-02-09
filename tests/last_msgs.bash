#!/bin/bash

curl --request GET \
    --url http://localhost:8080/messages \
    -H 'Authorization: '$TOKEN'' \
    --header 'User-Agent: insomnia/10.3.0'
