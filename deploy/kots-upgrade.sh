#!/bin/bash

set -e

while [ "$1" != "" ]; do
    _param="$(echo "$1" | cut -d= -f1)"
    _value="$(echo "$1" | grep '=' | cut -d= -f2-)"
    case $_param in
        registry)
            REGISTRY="$_value"
            ;;
        namespace)
            NAMESPACE="$_value"
            ;;
        *)
            echo >&2 "Error: unknown parameter \"$_param\""
            exit 1
            ;;
    esac
    shift
done

if [ -n "$REGISTRY" ]
then
    /kots admin-console push-images docker.io $REGISTRY -n $NAMESPACE
fi

/kots admin-console upgrade -n $NAMESPACE
