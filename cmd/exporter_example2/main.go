/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/xid"
	"gitlab.com/xerra/common/go-tcpinfo/pkg/exporter"
)

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <webroot>\n", os.Args[0])
		os.Exit(1)
	}

	webRoot := os.Args[1]

	if _, err := os.Stat(webRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Webroot %s does not exist\n", os.Args[1])
		os.Exit(2)
	}

	fs := http.FileServer(http.Dir(webRoot))
	http.Handle("/files/", http.StripPrefix("/files", fs))

	collector := exporter.NewTCPInfoCollector(
		"tcpinfo",
		[]string{"id", "remote_host"},
		prometheus.Labels{
			"app":      "exporter_example2",
			"hostname": hostname,
		},
		func(err error) {
			fmt.Println(err)
		},
	)

	prometheus.MustRegister(&collector)

	server := http.Server{
		Addr: ":18080",
		ConnState: func(conn net.Conn, state http.ConnState) {
			fmt.Printf("In goroutine for ConnState=%v/%d, net.Conn[%v -> %v]\n", state, int(state), conn.LocalAddr(), conn.RemoteAddr())
			switch state {
			case http.StateNew:
				collector.Add(conn, []string{xid.New().String(), conn.RemoteAddr().String()})
			case http.StateClosed:
				collector.Remove(conn)
			}
		},
	}

	http.Handle("/metrics", promhttp.Handler())
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
