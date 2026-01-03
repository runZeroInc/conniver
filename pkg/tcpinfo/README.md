# tcpinfo

This Go module provides an interface for obtaining `TCP_INFO` and similar data from sockets.
It is particularly useful for monitoring and diagnosing TCP connection problems.


## Linux Support

### Features

* **Detailed TCP Information:** Retrieves the `tcp_info` struct from the Linux kernel, providing a wealth of information about TCP connections.
* **Kernel Version Aware:** Gracefully handles different kernel versions, marking fields as unavailable on older kernels.
* **Prometheus Exporter:** Includes a ready-to-use Prometheus collector for easy integration into monitoring stacks.

### Why this library exists
The Linux `TCP_INFO` socket option is a powerful diagnostic tool, but its underlying data structure slowly evolves
with the kernel. This creates a significant challenge for developers who need to write applications that are both
comprehensive in the data they gather and portable across different Linux distributions and kernel versions. A naive
implementation may fail to compile, crash at runtime, or silently miss available data when moving between systems. This
library solves that problem.

### Solving the kernel API problem
This module provides a robust and reliable way to access TCP_INFO data through a unique two-pronged approach:

1. Completeness: It offers a comprehensive Go struct that maps to the full `tcp_info` data available in modern Linux
kernels (up to v6.7 and beyond), ensuring you have access to the latest metrics for advanced congestion control 
algorithms and TCP features.
2. Robustness: Unlike other libraries that are either outdated or rely on brittle compile-time checks, this module
detects the host's kernel version at runtime. It then intelligently populates only the fields that are genuinely
supported by the running kernel, guaranteeing that a single, statically-compiled binary works correctly and safely
across a wide range of Linux systems.

### A safe and unambiguous API

A key feature of this library is its commitment to API safety. When a `TCP_INFO` field is not supported by the host
kernel, its corresponding field in the returned Go struct is not simply left as a zero-value (e.g., 0 or an empty
string), which could be ambiguous. Instead, all fields added after kenel 2.6.2 have "nullable types" - a small struct
with a `Value` and `Valid` member, where the latter is explicitly set to true when supported. This allows your code
to reliably and easily distinguish between a metric that is truly zero and a metric that is unavailable on the host
system, preventing subtle bugs and leading to more robust applications.

### Comparison to Alternatives
- [golang.org/x/sys/unix](https://pkg.go.dev/golang.org/x/sys/unix#GetsockoptTCPInfo): Provides the low-level
primitives but offers no versioning safety, leaving the developer to manage the complexity and risk.
- [github.com/m-lab/tcp-info](https://github.com/m-lab/tcp-info): A powerful, actively maintained tool for system-wide
polling using the netlink interface. It is designed for a different use case (mass data collection) and its API
implicitly zero-values unsupported fields.
- [github.com/mikioh/tcpinfo](https://github.com/mikioh/tcpinfo): An earlier attempt to solve this problem that is
now unmaintained and significantly out-of-date. It uses a less flexible compile-time versioning strategy and lacks
support for most modern TCP features.

This library is the modern, safe, and flexible choice for developers needing to perform per-socket `TCP_INFO` diagnostics
in Go.

### Usage

The `tcpinfo.GetTCPInfo()` function provides direct access to the TCP socket information. Here is a simple example
that establishes a TCP connection, retrieves its `TCP_INFO`, and prints some of the retrieved statistics:

```go
package main

import (
	"fmt"
	"github.com/higebu/netfd"
	"github.com/runZeroInc/conniver/pkg/tcpinfo"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "google.com:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fd := netfd.GetFdFromConn(conn)
	if err != nil {
		panic(err)
	}

	tcpInfo, err := tcpinfo.GetTCPInfo(fd)
	if err != nil {
		panic(err)
	}

	jb, _ := json.MarshalIndent(tcpInfo, "  ")
	fmt.Printf("%s\n", string(jb))
}
```

Example output:
```
RTT: 20928, RTTVar: 10464
MinRTT: 20928
```

### Installation

To use this module in your project, install it with `go get`:

```bash
go get github.com/runZeroInc/conniver
```