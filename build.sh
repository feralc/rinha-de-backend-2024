#!/bin/bash

TAG="${1:-latest}"
PUSH_FLAG=false

# Parse command line options
while [[ $# -gt 0 ]]; do
    case $2 in
        --push)
            PUSH_FLAG=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

echo "building images with tag $TAG"

(
    docker build -t felipealcantara/rinha-de-backend-2024:$TAG .
) &

(
    docker build -f Dockerfile.lb -t felipealcantara/rinha-de-backend-2024-lb:$TAG .
) &

wait

echo "finish building images with tag $TAG"

if [ "$PUSH_FLAG" = true ]; then
    echo "pushing images to registry..."
    docker push felipealcantara/rinha-de-backend-2024:$TAG
    docker push felipealcantara/rinha-de-backend-2024-lb:$TAG
fi