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
	"io"
	"log"
	"net/url"
	"os/exec"
)

type Thumbnailer interface {
	// uri is the url to the original blobRef
	Command(uri url.URL) (prog string, args []string)
}

type FfmpegThumbnail struct{}

// http://superuser.com/questions/538112/meaningful-thumbnails-for-a-video-using-ffmpeg
// ffmpeg -ss 3 -i input.mp4 -vf "select=gt(scene\,0.4)" -frames:v 5 -vsync vfr fps=fps=1/600 out%02d.jpg

func (f FfmpegThumbnail) Command(uri url.URL) (string, []string) {
	return "ffmpeg", []string{
		"-i", uri.String(),
		"-vf", "thumbnail",
		"-frames:v", "1",
		"-f", "image2pipe",
		"pipe:1",
	}
}

type LogWriter struct{}

func (_ LogWriter) Write(data []byte) (int, error) {
	log.Print(string(data))
	return len(data), nil
}

func BuildCmd(tn Thumbnailer, uri url.URL, out io.Writer) *exec.Cmd {
	prog, args := tn.Command(uri)
	command := exec.Command(prog, args...)
	command.Stderr = LogWriter{}
	command.Stdout = out
	return command
}
