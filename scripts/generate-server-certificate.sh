#!/bin/bash

printf "Generates a self-signed TLS certificate.\n\n"

name="mosi"
printf "Certificate name [$name]: "
read s
if [ "$s" != "" ]; then name="$s"; fi

organization="Mosi Docker Registry"
printf "Organization [$organization]: "
read s
if [ "$s" != "" ]; then organization="$s"; fi

hosts=
while [ true ]; do
	printf "Server host names (comma-separated): "
	read hosts
	IFS=',' read -r -a hosts <<< "$hosts"
	if [ ${#hosts[@]} -gt 0 ]; then
		break
	fi
done

ips=
while [ true ]; do
	printf "Server IP addresses (comma-separated): "
	read ips
	IFS=',' read -r -a ips <<< "$ips"
	if [ ${#ips[@]} -gt 0 ]; then
		break
	fi
done


printf "\n%s\n%s\n" "Ready to generate" "--------------------------------------------------------------------------------"
printf "cert file    : $name.crt\n"
printf " key file    : $name.key\n"
printf "organization : $organization\n"
printf "       hosts : "

string=
for i in "${!hosts[@]}"; do
	s=$(echo ${hosts[$i]}) # remove leading/trailing spaces
	printf "$s "
	if [ "$string" != "" ]; then string+=","; fi
	string+="DNS:$s"
done
printf "\n"
printf "         IPs : "
for i in "${!ips[@]}"; do
	s=$(echo ${ips[$i]}) # remove leading/trailing spaces
	printf "$s "
	if [ "$string" != "" ]; then string+=","; fi
	string+="IP:$s"
done
printf "\n"

printf "\nPress ENTER to generate the certificate "
read s

echo "[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no
[req_distinguished_name]
O = $organization
[v3_req]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer
basicConstraints = CA:TRUE
keyUsage = keyCertSign, dataEncipherment, keyEncipherment, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = $string" > $name.cfg

# PEM
openssl req -x509 -nodes -days 36500 -newkey rsa:2048 -keyout $name.key -out $name.crt -config $name.cfg -extensions 'v3_req'
rm $name.cfg

# P12
#openssl pkcs12 -export -in $name.crt -inkey $name.key -out $name.p12 -passout pass:mike