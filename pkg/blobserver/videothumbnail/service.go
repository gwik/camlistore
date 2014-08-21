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
	"net/http"
	"net/url"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/netutil"
)

// Configuration
var DefaultThumbnailer Thumbnailer = FfmpegThumbnail{}

//TODO(gwik) handle concurrence
type ThumbnailService struct {
	thumbnailer Thumbnailer
	Timeout     time.Duration
}

func (ts ThumbnailService) Generate(ref blob.Ref, fetcher blob.Fetcher, writer io.Writer) error {
	listener, err := netutil.ListenOnLocalRandomPort()
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
	command := BuildCmd(ts.thumbnailer, uri, writer)
	err = command.Start()
	defer func() {
		if !command.ProcessState.Exited() {
			command.Process.Kill()
		}
	}()
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

	select {
	case <-done:
		if command.ProcessState.Success() {
			return nil
		}
		return errors.New("Thumbnail subprocess failed.")
	case err := <-errChan:
		return err
	case <-time.After(ts.Timeout):
		return errors.New("Timeout.")
	}
}
