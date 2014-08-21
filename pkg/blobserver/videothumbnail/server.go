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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"camlistore.org/pkg/blob"
)

// Configuration
var Thumbnail Thumbnailer = FfmpegThumbnail{}

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

func MakeThumbnail(ref blob.Ref, fetcher blob.Fetcher, writer io.Writer) error {
	listener, port, err := ListenOnLocalRandomPort()
	if err != nil {
		return err
	}
	defer listener.Close()

	uri := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", port),
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
		listener.Close()
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

/*

TODO

- Build HTTP server and deal with shutdown (Close()? on the listener)
- Communicate port and PID of process to the Handler in order to
  check them with ident.

*/
