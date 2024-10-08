/*
Copyright (c) 2024 Lasse Østerild

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	caFile   = "/certs/mongodb-ca-cert"
	certFile = "/certs/mongodb.pem"
)

var (
	// set during build
	version = "0.0.0"
	commit  = "dev"
	date    = "dev"
	builtBy = "manually"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var tlsConfig *tls.Config

	cmddbname := flag.String("db", "admin", "database name")
	cmdhost := flag.String("host", "", "hostname (default: \"127.0.0.1\")")
	cmdtls := flag.Bool("tls", false, "use mTLS")
	cmdhello := flag.Bool("hello", false, "readiness & startup probe")
	cmdping := flag.Bool("ping", false, "liveness probe")
	cmdversion := flag.Bool("version", false, "version")

	flag.Parse()

	if *cmdversion {
		fmt.Printf("mongodb-k8s-probe %s, commit %s, date %s, built by: %s\n", version, commit, date, builtBy)
		return nil
	}

	// cmdhello and cmdping are mutually exclusive
	if (*cmdhello && *cmdping) || (!*cmdhello && !*cmdping) {
		flag.PrintDefaults()
		return errors.New("hello and ping are mutually exclusive")
	}

	// hostname order: command-line, env var, 127.0.0.1
	if len(*cmdhost) == 0 {
		*cmdhost = os.Getenv("HOSTNAME")
		if len(*cmdhost) == 0 {
			*cmdhost = "127.0.0.1"
		}
	}

	mport := os.Getenv("MONGODB_PORT_NUMBER")
	if len(mport) == 0 {
		mport = "27017"
	}

	if *cmdtls {
		// Loads CA certificate file
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return err
		}
		// Loads certificate file
		clientCert, err := os.ReadFile(certFile)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return errors.New("certificate CA file must be in PEM format")
		}
		// Loads client certificate files
		cert, err := tls.X509KeyPair(clientCert, clientCert)
		if err != nil {
			return err
		}

		tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // #nosec G402
			RootCAs:            caCertPool,
			Certificates:       []tls.Certificate{cert},
			MinVersion:         tls.VersionTLS13,
		}
	}

	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().
		ApplyURI(fmt.Sprintf("mongodb://%s:%s", *cmdhost, mport)).
		SetServerAPIOptions(serverAPI).
		SetAppName(fmt.Sprintf("mongodb-k8s-probe %s", version)).
		SetTLSConfig(tlsConfig).
		SetDirect(true).
		SetConnectTimeout(10 * time.Second)

	// Create a new client and connect to MongoDB
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return err
	}

	// make sure we don't forget to disconnect from MongoDB
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	if *cmdhello {
		// Send a hello and confirm we're ok'ish
		var result bson.M
		if err := client.Database(*cmddbname).RunCommand(context.TODO(), bson.D{{Key: "hello", Value: 1}}).Decode(&result); err != nil {
			return err
		}

		if !(result["isWritablePrimary"].(bool) || result["secondary"].(bool)) {
			return errors.New("not ready")
		}
	} else if *cmdping {
		// Send a ping to confirm a successful connection
		// if err := client.Database(*cmddbname).RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}); err != nil {
		if err := client.Ping(context.TODO(), nil); err != nil {
			return errors.New("not alive")
		}
	}

	return nil
}
