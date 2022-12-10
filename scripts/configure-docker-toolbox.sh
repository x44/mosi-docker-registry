#!/bin/bash

# Configures the Docker Toolbox VM for usage with the Mosi repository.
# 1) Maps the Mosi host name(s) to IP address(es)
# 2) Stores the Mosi TLS certificate(s)

# Note, that the Docker Toolbox VM must be running!

# Get the VM IP
vm_ip=$(docker-machine ip)
if [ $? -ne 0 ]; then
	echo "VM NOT RUNNING?"
	exit
fi
echo "     VM IP: $vm_ip"

# Find the VM's SSH key
vm_key=$HOME/.docker/machine/machines/default/id_rsa
if [ ! -f "$vm_key" ]; then
	echo "VM SSH KEY NOT FOUND"
	exit
fi
echo "VM SSH KEY: $vm_key"


ips=()
hosts=()
i=1
while [ true ]; do
	printf "\nRepository %d\n" $i
	printf "Enter repository IP address or leave blank when done: "
	read ip
	if [ "$ip" == "" ]; then break; fi
	printf "Enter repository host name or leave blank when done: "
	read host
	if [ "$host" == "" ]; then break; fi
	ips+=("$ip")
	hosts+=("$host")
	i=$((i+1))
done

certs=()
i=1
while [ true ]; do
	printf "\nCertificate %d\n" $i
	while [ true ]; do
		printf "Enter certificate file name or leave blank when done: "
		read cert
		if [ "$cert" == "" ]; then break 2; fi
		if [ -f "$cert" ]; then
			certs+=("$cert")
			break;
		fi
		printf "File not found!\n"
	done
	i=$((i+1))
done

printf "\n"

hosts_str=
if [ ${#ips[@]} -gt 0 ]; then
	printf "Hosts to configure\n"
	printf "%s\n" "----------------------------------------"
	for i in "${!ips[@]}"; do
		printf "%-16s%s\n" "${ips[$i]}" "${hosts[$i]}"
		if [ "$hosts_str" != "" ]; then hosts_str+="&&"; fi
		hosts_str+="echo ${ips[$i]} ${hosts[$i]} >> /etc/hosts"
	done
	printf "\n"
fi

if [ ${#certs[@]} -gt 0 ]; then
	printf "\nCertificates to install\n"
	printf "%s\n" "----------------------------------------"
	for i in "${!certs[@]}"; do
		printf "${certs[$i]}\n"
	done
	printf "\n"
fi

if [ ${#ips[@]} -eq 0 ] && [ ${#certs[@]} -eq 0 ]; then
	printf "Nothing to do"
	exit
fi

printf "Press ENTER to begin "
read s

printf "\n"

if [ ${#ips[@]} -gt 0 ]; then
	printf "Configuring hosts...\n"
	ssh -i $vm_key docker@$vm_ip "echo -e '#!/bin/bash\nsleep 5\nsudo -i\n$hosts_str\nexit' > bootlocal.sh&&sudo cp bootlocal.sh /var/lib/boot2docker/&&sudo chmod +x /var/lib/boot2docker/bootlocal.sh"
	printf "\n"
fi

if [ ${#certs[@]} -gt 0 ]; then
	printf "Installing certificates...\n"
	ssh -i $vm_key docker@$vm_ip "mkdir -p ~/certs_tmp&&rm -f ~/certs_tmp/*"
	for i in "${!certs[@]}"; do
		scp -i $vm_key "${certs[$i]}" docker@$vm_ip:"~/certs_tmp/"$(basename "${certs[$i]}")
	done
	ssh -i $vm_key docker@$vm_ip "sudo mkdir -p /var/lib/boot2docker/certs&&sudo cp ~/certs_tmp/* /var/lib/boot2docker/certs"
	printf "\n"
fi

printf "Done. Press ENTER to reboot the VM "
read s

ssh -i $vm_key docker@$vm_ip "sudo reboot now"
