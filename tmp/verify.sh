#!/bin/bash
DIR=./tmp

tls_json=$(kubectl get cm -n test example-config -ojsonpath="{.data.JSON}")

orig_ca_pem=$(echo "$tls_json" | jq -r '.ca.Cert' | tr "\n" " ")
orig_cert_pem=$(echo "$tls_json" | jq -r '.cert.Cert' | tr "\n" " ")
orig_key_pem=$(echo "$tls_json" | jq -r '.cert.Key' | tr "\n" " ")

# Create Files directly from the JSON, since it maintains the formatting characters
echo "$tls_json" | jq -r '.ca.Cert' > "$DIR/orig_ca.pem"
echo "$tls_json" | jq -r '.cert.Cert' > "$DIR/orig_cert.pem"
echo "$tls_json" | jq -r '.cert.Key' > "$DIR/orig_key.pem"

config_ca_pem=$(kubectl get cm -n test example-config -ojsonpath="{.data.CA}")
config_cert_pem=$(kubectl get cm -n test example-config -ojsonpath="{.data.CERT}")
config_key_pem=$(kubectl get cm -n test example-config -ojsonpath="{.data.KEY}")

echo "$config_ca_pem"  > "$DIR/cm_ca.pem"
echo "$config_cert_pem" > "$DIR/cm_cert.pem"
echo "$config_key_pem" > "$DIR/cm_key.pem"

function compare() {
    local orig=$(echo $1)
    local config=$(echo $2)
    local msg=$3
    if [ "$orig" != "$config" ]; then
        echo "ERROR: $msg does not match"
        diff <(echo "$orig") <(echo "$config")
    else 
        echo "OK: json and cm match for $msg "
    fi
}

function verify_chain() {
    local ca_pem=$1
    local cert_pem=$2
    local msg=$3

    if openssl verify -CAfile "$ca_pem" "$cert_pem"  &> /dev/null; then
        echo "OK: cert chain for $msg is valid"
    else
        echo "ERROR: Cert chain for $msg  is not valid"
    fi
}

function verify_keypair() {
    local cert_pem=$1
    local key_pem=$2
    local msg=$3

    local key_md5=$(openssl rsa -modulus -noout -in "$key_pem" | openssl md5)
    local cert_md5=$(openssl x509 -modulus -noout -in "$cert_pem" | openssl md5)
    
    if [ "$key_md5" != "$cert_md5" ]; then
        echo "ERROR: cert and key for $msg do not match"
    else 
        echo "OK: key and cert for $msg  mod match"
    fi

}

compare "$orig_ca_pem" "$config_ca_pem" "CA"
compare "$orig_cert_pem" "$config_cert_pem" "CERT"
compare "$orig_key_pem" "$config_key_pem" "KEY"

verify_chain "$DIR/orig_ca.pem" "$DIR/orig_cert.pem" "original JSON"
# verify_chain "$DIR/cm_ca.pem" "$DIR/cm_cert.pem" "config map value"   # doesn't work because of formatting

verify_keypair "$DIR/orig_cert.pem" "$DIR/orig_key.pem" "original JSON"
# verify_keypair "$DIR/cm_cert.pem" "$DIR/cm_key.pem" "config map value" # doesn't work because of formatting



