package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"mosi-docker-repo/pkg/cmdline"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type args struct {
	name  *string
	dir   *string
	hosts *string
	ips   *string
}

func write(buf *bytes.Buffer, fn string) error {
	dir := filepath.Dir(fn)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(buf.String())
	return err
}

func publicKey(privateKey interface{}) interface{} {
	switch t := privateKey.(type) {
	case *rsa.PrivateKey:
		return &t.PublicKey
	}
	return nil
}

func generate(hosts, ips []string, crtFile, keyFile string) error {
	valid := true
	nips := []net.IP{}
	for _, ip := range ips {
		nip := net.ParseIP(ip)
		if nip != nil {
			nips = append(nips, nip)
		} else {
			fmt.Printf("Invalid IP: %s\n", ip)
			valid = false
		}
	}
	if !valid {
		return errors.New("invalid input")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mosi Docker Repository"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 36500),

		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IsCA: true,
	}
	template.DNSNames = append(template.DNSNames, hosts...)
	template.IPAddresses = append(template.IPAddresses, nips...)

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	if err != nil {
		return err
	}
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	err = write(out, crtFile)
	if err != nil {
		return err
	}
	out.Reset()
	pem.Encode(out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	err = write(out, keyFile)
	if err != nil {
		return err
	}
	return nil
}

func inputArgs(certsDefaultDir string, a *args) {
	cmdline.InputString("Certificate name", "mosi", &a.name)
	cmdline.InputString("Output directory", certsDefaultDir, &a.dir)
	cmdline.InputString("Server host names (comma or space-separated)", "mosi", &a.hosts)
	cmdline.InputString("Server IP addresses (comma or space-separated)", "127.0.0.1", &a.ips)
}

func exists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

func main() {
	fmt.Printf("This tool generates a self-signed certificate for server authentication.\n")
	fmt.Printf("Run this tool without arguments for interactive mode. Run with -h for help on arguments.\n")

	certsDefaultDir := "_dev/certs"
	if filepath.Base(filepath.Dir(os.Args[0])) == "tools" {
		certsDefaultDir = "../certs"
	}

	a := args{}
	if len(os.Args) == 1 {
		inputArgs(certsDefaultDir, &a)
		fmt.Printf("\n")
	} else {
		a.name = flag.String("name", "mosi", "Name of the certificate. Output files will be <name>.crt and <name>.key")
		a.dir = flag.String("dir", certsDefaultDir, "Output directory")
		a.hosts = flag.String("hosts", "mosi", "Server host names (comma- or space-separated)")
		a.ips = flag.String("ips", "127.0.0.1", "Server IP addresses (comma- or space-separated)")
		flag.Parse()
	}

	crtFile, _ := filepath.Abs(filepath.Join(*a.dir, *a.name+".crt"))
	keyFile, _ := filepath.Abs(filepath.Join(*a.dir, *a.name+".key"))
	hosts := cmdline.SplitByCommaOrSpaceAndTrim(*a.hosts)
	ips := cmdline.SplitByCommaOrSpaceAndTrim(*a.ips)

	fmt.Printf("Ready to generate the certificate\n")
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("Certificate file : %s\n", crtFile)
	fmt.Printf("Key file         : %s\n", keyFile)
	fmt.Printf("Host names       : %s\n", strings.Join(hosts, ", "))
	fmt.Printf("IP addresses     : %s\n", strings.Join(ips, ", "))

	if exists(crtFile) {
		fmt.Printf("\nFile exists and will be overwritten: %s\n", crtFile)
	}
	if exists(keyFile) {
		fmt.Printf("\nFile exists and will be overwritten: %s\n", keyFile)
	}

	fmt.Printf("\nPress ENTER to generate the certificate ")
	fmt.Scanln()

	err := generate(hosts, ips, crtFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
}
