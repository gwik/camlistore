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

/*
Package videothumbnail generates image thumbnails from videos.

(*Service).Generate spawns an HTTP server listening on a local random
port to serve the video to an external program (see Thumbnailer interface).
The external program is expected to output the thumbnail image on its
standard output.

The default implementation uses ffmpeg.

See ServiceFromConfig for accepted configuration.
*/
package videothumbnail

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/jsonconfig"
	"camlistore.org/pkg/netutil"
	"camlistore.org/pkg/syncutil"
)

// A Service controls the generation of video thumbnails.
type Service struct {
	thumbnailer Thumbnailer
	// Timeout is the maximum duration for the thumbnailing subprocess execution.
	timeout time.Duration
	// Limit number of running subprocesses.
	gate *syncutil.Gate
}

// ServiceFromConfig builds a new Service from configuration.
// Example expected configuration object (all keys are optional) :
// {
//   // command defaults to FFmpegThumbnailer and `$uri` is replaced by
//   // the real value at runtime.
//   "command": ["/opt/local/bin/ffmpeg", "-i", "$uri", "pipe:1"],
//   // Maximun number of milliseconds for running the thumbnailing subprocess.
//   "timeout": 2000,
//   // Maximum number of thumbnailing subprocess running at same time.
//   "maxProcs": 5
// }
func ServiceFromConfig(conf jsonconfig.Obj) *Service {
	th := thumbnailerFromConfig(conf)
	timeout := conf.OptionalInt("timeout", 5000)
	maxProc := conf.OptionalInt("maxProcs", 5)

	conf.Validate()

	return NewService(th, time.Millisecond*time.Duration(timeout), maxProc)
}

// NewService builds a new Service.
func NewService(th Thumbnailer, timeout time.Duration, maxProc int) *Service {
	return &Service{
		thumbnailer: th,
		timeout:     timeout,
		gate:        syncutil.NewGate(maxProc),
	}
}

var errTimeout = errors.New("Timeout.")

// Generate generates a thumbnail and write it to writer.
func (s *Service) Generate(
	ref blob.Ref, fetcher blob.Fetcher, writer io.Writer) error {

	s.gate.Start()
	defer s.gate.Done()

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
	cmd := buildCmd(s.thumbnailer, uri, writer)
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()
	go func() {
		cmd.Wait()
		done <- true
	}()

	errChan := make(chan error)
	go func() {
		err := http.Serve(listener,
			createVideothumbnailHandler(ref, fetcher))
		errChan <- err
	}()

	select {
	case <-done:
		if cmd.ProcessState.Success() {
			return nil
		}
		return errors.New("Thumbnail subprocess failed.")
	case err := <-errChan:
		return err
	case <-time.After(s.timeout):
		return errTimeout
	}
}

// Available tells whether the service can run.
func (s *Service) Available() error {
	prog, _ := s.thumbnailer.Command(url.URL{Path: "/"})
	if _, err := exec.LookPath(prog); err != nil {
		return err
	}
	return nil
}
