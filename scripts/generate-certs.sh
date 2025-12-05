#!/bin/bash
set -e

# Generate TLS certificates for lnmonja

CA_DIR="./certs"
SERVER_DIR="./certs/server"
CLIENT_DIR="./certs/client"

mkdir -p "$CA_DIR" "$SERVER_DIR" "$CLIENT_DIR"

# Generate CA
openssl genrsa -out "$CA_DIR/ca.key" 4096
openssl req -x509 -new -nodes -key "$CA_DIR/ca.key" \
    -sha256 -days 3650 -out "$CA_DIR/ca.crt" \
    -subj "/C=US/ST=CA/L=SF/O=lnmonja/CN=lnmonja CA"

# Generate server certificate
openssl genrsa -out "$SERVER_DIR/server.key" 2048
openssl req -new -key "$SERVER_DIR/server.key" \
    -out "$SERVER_DIR/server.csr" \
    -subj "/C=US/ST=CA/L=SF/O=lnmonja/CN=lnmonja-server"

openssl x509 -req -in "$SERVER_DIR/server.csr" \
    -CA "$CA_DIR/ca.crt" -CAkey "$CA_DIR/ca.key" \
    -CAcreateserial -out "$SERVER_DIR/server.crt" \
    -days 365 -sha256

# Generate client certificate
openssl genrsa -out "$CLIENT_DIR/client.key" 2048
openssl req -new -key "$CLIENT_DIR/client.key" \
    -out "$CLIENT_DIR/client.csr" \
    -subj "/C=US/ST=CA/L=SF/O=lnmonja/CN=lnmonja-agent"

openssl x509 -req -in "$CLIENT_DIR/client.csr" \
    -CA "$CA_DIR/ca.crt" -CAkey "$CA_DIR/ca.key" \
    -CAcreateserial -out "$CLIENT_DIR/client.crt" \
    -days 365 -sha256

# Create combined PEM files
cat "$SERVER_DIR/server.crt" "$SERVER_DIR/server.key" > "$SERVER_DIR/server.pem"
cat "$CLIENT_DIR/client.crt" "$CLIENT_DIR/client.key" > "$CLIENT_DIR/client.pem"

echo "Certificates generated:"
echo "- CA: $CA_DIR/ca.crt"
echo "- Server: $SERVER_DIR/server.pem"
echo "- Client: $CLIENT_DIR/client.pem"