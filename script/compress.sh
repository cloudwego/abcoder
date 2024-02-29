#!/bin/bash

check_url_status() {
    url=$1
    while true; do
        echo "start to process:" $1
        response=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8888/repo_compress?repo="$url)

        # If the status code is 200, exit the script; otherwise, try again after waiting for 5 minutes
        if [ $response -eq 200 ]; then
            echo "Request to $url was successful. Exiting."
            break
        else
            echo "Request to $url failed with status $response. Retrying in 5 minutes..."
            sleep 300
        fi
    done
} 

# List of URLs
# urls=("hertz-contrib/logger/zap" "hertz-contrib/obs-opentelemetry" "hertz-contrib/sse" "hertz-contrib/websocket" "hertz-contrib/registry" "hertz-contrib/reverseproxy")

# urls=("cloudwego/kitex")

check_url_status $1
