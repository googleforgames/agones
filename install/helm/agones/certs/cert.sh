#!/usr/bin/env bash

echo "Generating certs..."
echo "Email should be: emailAddress=agones-discuss@googlegroups.com"
echo "Common Name should be: agones-controller-service.agones-system.svc"
openssl genrsa -out server.key 2048
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650

echo "caBundle:"
base64 -w 0 server.crt

echo "done"