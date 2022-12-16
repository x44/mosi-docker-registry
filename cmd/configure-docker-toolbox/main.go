package main

import (
	"bufio"
	"flag"
	"fmt"
	"mosi-docker-registry/pkg/terminal"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type args struct {
	hosts *string
	certs *string
}

type hostsEntry struct {
	ip   string
	name string
}

func run(cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	b, err := c.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ssh(msg, cmd string) {
	fmt.Printf("%s ... ", msg)
	s, err := run("docker-machine", "ssh", "default", cmd)
	if err != nil {
		fmt.Printf("FAILED!\n%v", err)
		os.Exit(1)
	}
	fmt.Printf("OK %s\n", s)
}

func createBootLocalScript(hosts *[]hostsEntry) {
	hosts_str := ""
	for i, e := range *hosts {
		if i > 0 {
			hosts_str += "&&"
		}
		hosts_str += "echo "
		hosts_str += e.ip
		hosts_str += " "
		hosts_str += e.name
		hosts_str += " >> /etc/hosts"
	}
	cmd := "echo -e '#!/bin/bash\nsleep 5\nsudo -i\n" + hosts_str + "\nexit' > bootlocal.sh&&sudo cp bootlocal.sh /var/lib/boot2docker/&&sudo chmod +x /var/lib/boot2docker/bootlocal.sh"
	ssh("Creating bootlocal.sh", cmd)
}

func installCertificate(fn string) {
	// docker-machine scp does not work (it resolves the docker VM to 127.0.0.1) so we use ssh to copy the cert
	name := filepath.Base(fn)
	f, err := os.Open(fn)
	if err != nil {
		fmt.Printf("failed to open %s\n%v", fn, err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	data := ""
	for scanner.Scan() {
		s := string(scanner.Bytes())
		data += s
		data += "\\n"
	}
	data = data[:len(data)-2]

	cmd := "echo -e '" + data + "' > " + name + "&&sudo mkdir -p /var/lib/boot2docker/certs&&sudo cp " + name + " /var/lib/boot2docker/certs/"
	ssh("Installing certificate "+name, cmd)
}

func installCertificates(fns *[]string) {
	for _, fn := range *fns {
		installCertificate(fn)
	}
}

func inputArgs(a *args) {
	terminal.InputString("\nEnter the hosts to add to the default /etc/hosts file on the Docker VM.\nLeave blank to not change the hosts configuration.\nFormat: <IP> <HOSTNAME>, <IP> <HOSTNAME>, ...\nExample: 192.168.0.1 myhost, 192.168.0.2 yourhost\nHosts", "", &a.hosts)
	terminal.InputString("\nEnter the certificates to install on the Docker VM.\nLeave blank to not add certificates now.\nFormat: <FILENAME>, <FILENAME>, ...\nExample: mycert.crt, certs/yourcert.crt\nCerts", "", &a.certs)
}

func main() {
	fmt.Printf("This tool adds additional host mappings and server certificates to the Docker Toolbox VM.\n")
	fmt.Printf("You can either\n")
	fmt.Printf("- Add host mappings AND server certificates in one go\n")
	fmt.Printf("- Add host mappings only (by ommitting the -certs argument or by not entering certs in interactive mode)\n")
	fmt.Printf("- Add server certificates only (by ommitting the -hosts argument or by not entering hosts in interactive mode)\n")
	fmt.Printf("Please note, that whenever you use this tool to add host mappings, you have to add ALL your host mappings again.\n")
	fmt.Printf("For example, if you call this tool once to add '1.2.3.4 myhost' and call it again to add '5.6.7.8 yourhost',\n")
	fmt.Printf("only '5.6.7.8 yourhost' will be in the VM's /etc/hosts file.\n")
	fmt.Printf("Further note, that this tool will not remove any certificates from the VM and\n")
	fmt.Printf("existing certificates with the same name will be overwritten without any warning.\n")
	fmt.Printf("Run this tool without arguments for interactive mode. Run with -h for help on arguments.\n")

	a := args{}

	if len(os.Args) == 1 {
		inputArgs(&a)
		fmt.Printf("\n")
	} else {
		a.hosts = flag.String("hosts", "", "Hosts to add to the default /etc/hosts file on the Docker VM.\nOmmit this argument to not change the hosts configuration.\nFormat: <IP> <HOSTNAME>, <IP> <HOSTNAME>, ...\nExample: -hosts \"192.168.0.1 myhost, 192.168.0.2 yourhost\"")
		a.certs = flag.String("certs", "", "Certificates to install on the Docker VM.\nOmmit this argument to not add certificates.\nFormat: <FILENAME>, <FILENAME>, ...\nExample: -certs \"mycert.crt, certs/yourcert.crt\"")
		flag.Parse()
	}

	hosts := []hostsEntry{}
	certs := []string{}

	if a.hosts != nil && *a.hosts != "" {
		argsHostsList := strings.Split(*a.hosts, ",")
		for _, argsHost := range argsHostsList {
			argsHost = strings.TrimSpace(argsHost)
			ipAndHost := strings.Split(argsHost, " ")
			if len(ipAndHost) != 2 {
				fmt.Printf("Invalid 'hosts' input: '%s'\n", *a.hosts)
			}
			hosts = append(hosts, hostsEntry{ip: strings.TrimSpace(ipAndHost[0]), name: strings.TrimSpace(ipAndHost[1])})
		}
	}
	if a.certs != nil && *a.certs != "" {
		argsCertsList := strings.Split(*a.certs, ",")
		for _, argsCert := range argsCertsList {
			certs = append(certs, strings.TrimSpace(argsCert))
		}
	}

	if len(hosts) == 0 && len(certs) == 0 {
		fmt.Printf("Nothing to do\n")
		return
	}

	if len(hosts) > 0 {
		fmt.Printf("\nHosts to add\n")
		fmt.Printf("--------------------------------------------------------------------------------\n")
		for _, host := range hosts {
			fmt.Printf("%-16s%s\n", host.ip, host.name)
		}
	}
	if len(certs) > 0 {
		fmt.Printf("\nCertificates to install\n")
		fmt.Printf("--------------------------------------------------------------------------------\n")
		for _, cert := range certs {
			fmt.Printf("%s\n", cert)
		}
	}

	fmt.Printf("\nPress ENTER to begin ")
	fmt.Scanln()
	fmt.Printf("\n")

	createBootLocalScript(&hosts)
	installCertificates(&certs)

	fmt.Printf("\nDone. Press ENTER to reboot the VM ")
	fmt.Scanln()
	ssh("Rebooting", "sudo reboot now")
}
