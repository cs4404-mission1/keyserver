package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	key                 string
	caCertFile          = flag.String("ca-cert", "ca-crt.pem", "CA certificate")
	serverCertFile      = flag.String("server-cert", "ca-web-crt.pem", "Server certificate")
	serverKeyFile       = flag.String("server-key", "ca-web-key.pem", "Server key")
	authorizedClientSAN = flag.String("authorized-san", "api.internal", "Authorized client SAN")
	listen              = flag.String("listen", "172.16.10.1:443", "Listen address")
)

func main() {
	flag.Parse()

	caCert, err := os.ReadFile(*caCertFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	server := &http.Server{
		Addr: *listen,
		TLSConfig: &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
			VerifyConnection: func(state tls.ConnectionState) error {
				if len(state.PeerCertificates) > 0 && state.PeerCertificates[0].DNSNames[0] == *authorizedClientSAN {
					return nil
				} else {
					return fmt.Errorf("invalid client certificate")
				}
			},
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(key))
	})

	// Generate random 256bit key
	k := make([]byte, 32)
	if _, err = rand.Read(k); err != nil {
		log.Fatal(err)
	}
	key = fmt.Sprintf("%x", k)
	log.Printf("Generated key: %s", key)

	log.Printf("Starting server on %s", server.Addr)
	log.Fatal(server.ListenAndServeTLS(*serverCertFile, *serverKeyFile))
}
