#!/bin/bash

if [ "$DEBUG_KOTSADM" = "1" ]; then
    dlv --listen=:30001 --continue --headless=true --api-version=2 --accept-multiclient exec /kotsadm api
else
    /kotsadm api
fi
