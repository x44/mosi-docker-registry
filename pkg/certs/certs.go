package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/filesys"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// https://stackoverflow.com/a/37382208
func getOutboundIP() *net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return &localAddr.IP
}

func GetDefaultIPs() []string {
	ips := make([]string, 1)
	ips[0] = "127.0.0.1"
	outboundIP := getOutboundIP()
	if outboundIP != nil {
		ips = append(ips, outboundIP.String())
	}
	return ips
}

func publicKey(privateKey interface{}) interface{} {
	switch t := privateKey.(type) {
	case *rsa.PrivateKey:
		return &t.PublicKey
	}
	return nil
}

func Generate(hosts, ips []string, crtFile, keyFile string) error {
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
			Organization: []string{"Mosi Docker Registry"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 36500),

		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
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
	_, err = filesys.WriteBuffer(crtFile, out)
	if err != nil {
		return err
	}
	out.Reset()
	pem.Encode(out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	_, err = filesys.WriteBuffer(keyFile, out)
	if err != nil {
		return err
	}
	return nil
}

func GenerateDefault() error {
	crtFile := config.TlsCrtFile()
	keyFile := config.TlsKeyFile()

	if len(crtFile) == 0 || len(keyFile) == 0 {
		return nil
	}

	host := config.ServerHost()
	hosts := []string{host}
	ips := GetDefaultIPs()

	fn := filepath.Base(crtFile)
	fn = fn[:len(fn)-len(filepath.Ext(fn))]
	// use fmt here
	fmt.Printf("Creating default certificate for %s (%s): %s\n", host, strings.Join(ips, ", "), filepath.Join(filepath.Dir(crtFile), fn))
	return Generate(hosts, ips, crtFile, keyFile)
}
