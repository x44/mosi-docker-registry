package main

import (
	"flag"
	"fmt"
	"log"
	"mosi-docker-registry/pkg/certs"
	"mosi-docker-registry/pkg/filesys"
	"mosi-docker-registry/pkg/terminal"
	"os"
	"path/filepath"
	"strings"
)

type args struct {
	name  *string
	dir   *string
	hosts *string
	ips   *string
}

func inputArgs(defaultDir string, defaultIPs []string, a *args) {
	terminal.InputString("Certificate name", "mosi", &a.name)
	terminal.InputString("Output directory", defaultDir, &a.dir)
	terminal.InputString("Server host names (comma or space-separated)", "mosi", &a.hosts)
	terminal.InputString("Server IP addresses (comma or space-separated)", strings.Join(defaultIPs, ", "), &a.ips)
}

func main() {
	fmt.Printf("This tool generates a self-signed certificate for server authentication.\n")
	fmt.Printf("Run this tool without arguments for interactive mode. Run with -h for help on arguments.\n")

	defaultDir := "_dev/certs"
	if filepath.Base(filepath.Dir(os.Args[0])) == "tools" {
		defaultDir = "../certs"
	}

	defaultIPs := certs.GetDefaultIPs()

	a := args{}
	if len(os.Args) == 1 {
		inputArgs(defaultDir, defaultIPs, &a)
		fmt.Printf("\n")
	} else {
		a.name = flag.String("name", "mosi", "Name of the certificate. Output files will be <name>.crt and <name>.key")
		a.dir = flag.String("dir", defaultDir, "Output directory")
		a.hosts = flag.String("hosts", "mosi", "Server host names (comma- or space-separated)")
		a.ips = flag.String("ips", strings.Join(defaultIPs, ", "), "Server IP addresses (comma- or space-separated)")
		flag.Parse()
	}

	crtFile, _ := filepath.Abs(filepath.Join(*a.dir, *a.name+".crt"))
	keyFile, _ := filepath.Abs(filepath.Join(*a.dir, *a.name+".key"))
	hosts := terminal.SplitByCommaOrSpaceAndTrim(*a.hosts)
	ips := terminal.SplitByCommaOrSpaceAndTrim(*a.ips)

	fmt.Printf("Ready to generate the certificate\n")
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("Certificate file : %s\n", crtFile)
	fmt.Printf("Key file         : %s\n", keyFile)
	fmt.Printf("Host names       : %s\n", strings.Join(hosts, ", "))
	fmt.Printf("IP addresses     : %s\n", strings.Join(ips, ", "))

	if filesys.Exists(crtFile) {
		fmt.Printf("\nFile exists and will be overwritten: %s\n", crtFile)
	}
	if filesys.Exists(keyFile) {
		fmt.Printf("\nFile exists and will be overwritten: %s\n", keyFile)
	}

	fmt.Printf("\nPress ENTER to generate the certificate ")
	fmt.Scanln()

	err := certs.Generate(hosts, ips, crtFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
}
