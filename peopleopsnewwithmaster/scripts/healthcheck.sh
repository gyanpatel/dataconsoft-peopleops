#!/bin/sh
wget -q -O - http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1
