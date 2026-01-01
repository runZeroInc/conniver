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
	"time"

	"gitlab.com/xerra/common/go-tcpinfo/pkg/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func hallucinate() net.Conn {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 64)

		var n int
		for err == nil {
			n, err = conn.Read(buf)
			if err == nil {
				fmt.Print(string(buf[:n]))
			}
		}
	}()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		panic(err)
	}

	go func() {
		var err error
		for err == nil {
			_, err = conn.Write([]byte("badger, "))
			time.Sleep(time.Millisecond * 10)
		}
	}()

	return conn
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	conn := hallucinate()

	exp := exporter.NewTCPInfoCollector(
		"hallucination",
		nil,
		prometheus.Labels{
			"app":      "exporter_example1",
			"hostname": hostname,
		},
		func(err error) {
			panic(err)
		},
	)

	exp.Add(conn, nil)

	prometheus.MustRegister(&exp)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":18080", nil)
}
