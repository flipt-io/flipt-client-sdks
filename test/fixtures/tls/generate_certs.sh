#!/bin/bash

# Generate test certificates for self-signed certificate testing
set -e

# Create a simple CA
openssl genrsa -out ca.key 2048
openssl req -new -x509 -key ca.key -out ca.crt -days 3650 -subj "/C=US/ST=Test/L=Test/O=Flipt Test CA/CN=flipt-test-ca"

# Create server certificate
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=Test/L=Test/O=Flipt Test/CN=flipt-https"

# Create config for server cert with SAN
cat > server.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = flipt-https
DNS.2 = flipt
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF

# Sign server cert with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 3650 -extensions v3_req -extfile server.conf

# Clean up
rm server.csr server.conf ca.srl

echo "Generated certificates:"
echo "  ca.crt - Certificate Authority"
echo "  ca.key - CA private key"  
echo "  server.crt - Server certificate"
echo "  server.key - Server private key"