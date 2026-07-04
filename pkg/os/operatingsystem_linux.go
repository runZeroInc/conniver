// package os provides helper function to get the operating system
// name for different platforms.
package os

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

var (
	// file to use to detect if the daemon is running in a container
	proc1Cgroup = "/proc/1/cgroup"

	// file to check to determine Operating System
	etcOsRelease = "/etc/os-release"

	// used by stateless systems like Clear Linux
	altOsRelease = "/usr/lib/os-release"

	// marker files the runtime drops; on a cgroup v2 host /proc/1/cgroup is a
	// bare "0::/" and can't be told from the host by suffix alone
	dockerEnv    = "/.dockerenv"
	containerEnv = "/run/.containerenv"
)

// cgroup path fragments naming a container runtime, for the cgroup v2 case
// where the host check by suffix does not fire
var containerCgroupMarkers = [][]byte{
	[]byte("/docker"),
	[]byte("/lxc"),
	[]byte("/kubepods"),
	[]byte("/containerd"),
	[]byte("/machine"),
}

// GetOperatingSystem gets the name of the current operating system.
func GetOperatingSystem() (string, error) {
	if prettyName, err := getValueFromOsRelease("PRETTY_NAME"); err != nil {
		return "", err
	} else if prettyName != "" {
		return prettyName, nil
	}

	// If not set, defaults to PRETTY_NAME="Linux"
	// c.f. http://www.freedesktop.org/software/systemd/man/os-release.html
	return "Linux", nil
}

// GetOperatingSystemVersion gets the version of the current operating system, as a string.
func GetOperatingSystemVersion() (string, error) {
	return getValueFromOsRelease("VERSION_ID")
}

// parses the os-release file and returns the value associated with `key`
func getValueFromOsRelease(key string) (string, error) {
	osReleaseFile, err := os.Open(etcOsRelease)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		osReleaseFile, err = os.Open(altOsRelease)
		if err != nil {
			return "", err
		}
	}
	defer osReleaseFile.Close()

	var value string
	scanner := bufio.NewScanner(osReleaseFile)
	for scanner.Scan() {
		line := scanner.Text()
		if after, ok := strings.CutPrefix(line, key+"="); ok {
			value = after
			value = strings.Trim(value, `"' `) // remove leading/trailing quotes and whitespace
		}
	}

	return value, nil
}

// IsContainerized returns true if we are running inside a container.
func IsContainerized() (bool, error) {
	if fileExists(dockerEnv) || fileExists(containerEnv) {
		return true, nil
	}
	b, err := os.ReadFile(proc1Cgroup)
	if err != nil {
		return false, err
	}
	for line := range bytes.SplitSeq(b, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		// cgroup v1: a non-root path means containerized
		if !bytes.HasSuffix(line, []byte(":/")) && !bytes.HasSuffix(line, []byte(":/init.scope")) {
			return true, nil
		}
		// cgroup v2: single line whose path names a runtime
		for _, marker := range containerCgroupMarkers {
			if bytes.Contains(line, marker) {
				return true, nil
			}
		}
	}
	return false, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
