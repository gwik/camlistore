/*
Copyright 2014 The Camlistore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netutil

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// AwaitReachable tries to make a TCP connection to addr regularly.
// It returns an error if it's unable to make a connection before maxWait.
func AwaitReachable(addr string, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("%v unreachable for %v", addr, maxWait)
}

// HostPort takes a urlStr string URL, and returns a host:port string suitable
// to passing to net.Dial, with the port set as the scheme's default port if
// absent.
func HostPort(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("could not parse %q as a url: %v", urlStr, err)
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("url %q has no scheme", urlStr)
	}
	hostPort := u.Host
	if hostPort == "" || strings.HasPrefix(hostPort, ":") {
		return "", fmt.Errorf("url %q has no host", urlStr)
	}
	idx := strings.Index(hostPort, "]")
	if idx == -1 {
		idx = 0
	}
	if !strings.Contains(hostPort[idx:], ":") {
		if u.Scheme == "https" {
			hostPort += ":443"
		} else {
			hostPort += ":80"
		}
	}
	return hostPort, nil
}

// ListenOnLocalRandomPort returns a tcp listener on a local (see LoopbackIP) random port.
func ListenOnLocalRandomPort() (net.Listener, error) {
	ip, err := Localhost()
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: ip, Port: 0})
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Localhost returns the first address found when
// doing a lookup of "localhost".
func Localhost() (net.IP, error) {
	ips, err := net.LookupIP("localhost")
	if err != nil && len(ips) > 1 {
		// verify ip is a loopback ip ?
		return ips[0], nil
	}
	ip, err := loopbackIP()
	if err != nil {
		return nil, errors.New("IP lookup for localhost returned no result")
	}
	return ip, nil
}

const flagUpLoopback = net.FlagUp | net.FlagLoopback

// loopbackIP finds the first loopback IP address sniffing network interfaces.
func loopbackIP() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, inf := range interfaces {
		if inf.Flags&flagUpLoopback == flagUpLoopback {
			addrs, err := inf.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				// surprisingly this doesn't seem to be always the case.
				if err == nil && ip.IsLoopback() {
					return ip, nil
				}
			}
		}
	}
	return nil, errors.New("No loopback ip found.")
}
