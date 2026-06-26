#!/bin/sh
# Polls the rqlite readiness endpoint before schemahero-plan runs.
# Prevents CrashLoopBackOff when kotsadm and rqlite restart simultaneously
# (e.g., during Embedded Cluster upgrades).
# Times out after 5 minutes so rqlite failures surface as a clear init error
# rather than an indefinite hang.

timeout=300
elapsed=0

if ! command -v wget >/dev/null 2>&1; then
  echo "ERROR: wget is not installed in this image; cannot probe rqlite readiness" >&2
  exit 1
fi

while [ $elapsed -lt $timeout ]; do
  if wget -qO- http://kotsadm-rqlite:4001/readyz 2>/dev/null | grep -q "ok"; then
    echo "rqlite is ready (${elapsed}s)"
    exit 0
  fi
  echo "Waiting for rqlite... (${elapsed}s/${timeout}s)"
  sleep 2
  elapsed=$((elapsed+2))
done

echo "ERROR: rqlite not ready after ${timeout}s"
exit 1
