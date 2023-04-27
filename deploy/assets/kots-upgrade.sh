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
        "ensure-rbac" | "skip-rbac-check" | "strict-security-context" | "wait-duration" | "with-minio")
            INSTALL_PARAMS="$INSTALL_PARAMS --$_param=$_value"
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
    /kots admin-console copy-public-images $REGISTRY -n $NAMESPACE
fi

/kots admin-console upgrade -n $NAMESPACE $INSTALL_PARAMS
