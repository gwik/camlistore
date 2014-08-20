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

package videothumbnail

import (
	"net"
	"strconv"
	"strings"
)

// Listen on random port number and return listener, port and error
func ListenOnLocalRandomPort() (net.Listener, int, error) {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		return nil, 0, err
	}
	portStr := strings.TrimPrefix(l.Addr().String(), "127.0.0.1:")
	port, err := strconv.ParseInt(portStr, 10, 0)
	if err != nil {
		panic(err)
	}
	return l, int(port), nil
}
