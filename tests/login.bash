#!/bin/bash

curl --request POST \
    --url http://localhost:8080/login \
    --header 'Content-Type: application/json' \
    --header 'User-Agent: insomnia/10.3.0' \
    --data '{ "email": "user4@example.com", "password": "d0$4eba!23eae5a" }
'
