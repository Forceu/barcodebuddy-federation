#!/bin/bash
until ./BarcodeServer; do
    echo "Server 'BarcodeServer' crashed with exit code $?.  Respawning.." >&2
    sleep 5
done
