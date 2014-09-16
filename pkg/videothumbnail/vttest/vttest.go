package vttest

import (
	"net/url"
	"os/exec"
	"testing"

	"camlistore.org/pkg/videothumbnail"
)

func SkipThumbnailerNotAvailable(t *testing.T, tn videothumbnail.Thumbnailer) {
	prog, _ := tn.Command(url.URL{Path: "/"})
	if _, err := exec.LookPath(prog); err != nil {
		t.Skip(err)
	}
}
