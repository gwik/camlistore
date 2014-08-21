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
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"camlistore.org/pkg/blob"
)

// Configuration
var Thumbnail Thumbnailer = FfmpegThumbnail{}

func LoopbackInterfaceAddr() (net.Addr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, inf := range interfaces {
		if inf.Flags&(net.FlagLoopback|net.FlagUp) == net.FlagLoopback|net.FlagUp {
			addrs, err := inf.Addrs()
			if err != nil {
				continue
			}
			if len(addrs) > 0 {
				return addrs[0], nil
			}
		}
	}
	return nil, errors.New("No loopback interface found.")
}

// Listen on random port number and return listener, port and error
func ListenOnLocalRandomPort() (net.Listener, error) {
	addr, err := LoopbackInterfaceAddr()
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(addr.String())
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: ip, Port: 0})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func MakeThumbnail(ref blob.Ref, fetcher blob.Fetcher, writer io.Writer) error {
	listener, err := ListenOnLocalRandomPort()
	if err != nil {
		return err
	}
	defer listener.Close()

	uri := url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
		Path:   ref.String(),
	}

	done := make(chan bool)
	command := BuildCmd(Thumbnail, uri, writer)
	err = command.Start()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		err := http.Serve(listener,
			CreateVideothumbnailHandler(ref, fetcher, command.Process.Pid))
		errChan <- err
	}()

	go func() {
		command.Wait()
		done <- true
	}()

	defer func() {
		if !command.ProcessState.Exited() {
			command.Process.Kill()
		}
	}()

	select {
	case <-done:
		return nil
	case err := <-errChan:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}
