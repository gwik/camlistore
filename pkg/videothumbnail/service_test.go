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
	"bytes"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/magic"
	"camlistore.org/pkg/schema"
	"camlistore.org/pkg/test"
)

const testFilepath = "testdata/small.webm"

func storageAndBlobRef(t *testing.T) (blobserver.Storage, blob.Ref) {
	storage := new(test.Fetcher)
	inFile, err := os.Open(testFilepath)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := schema.WriteFileFromReader(storage, "small.webm", inFile)
	if err != nil {
		t.Fatal(err)
	}
	return storage, ref
}

func TestStorage(t *testing.T) {
	store, ref := storageAndBlobRef(t)
	fr, err := schema.NewFileReader(store, ref)
	if err != nil {
		t.Fatal(err)
	}
	inFile, err := os.Open(testFilepath)
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(inFile)
	if err != nil {
		t.Fatal(err)
	}
	bd, err := ioutil.ReadAll(fr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bd, data) {
		t.Error("expected to be the same")
	}
}

func TestMakeThumbnail(t *testing.T) {
	service := NewService(DefaultThumbnailer, 2*time.Second, 5)
	if err := service.Available(); err != nil {
		t.Skip(err)
	}
	store, ref := storageAndBlobRef(t)
	tmpFile, _ := ioutil.TempFile(os.TempDir(), "camlitest")
	defer tmpFile.Close()
	errMake := service.Generate(ref, store, tmpFile)

	if errMake != nil {
		t.Fatal(errMake)
	}

	tmpFile.Seek(0, 0)

	typ, _ := magic.MIMETypeFromReader(tmpFile)
	if typ != "image/png" {
		t.Errorf("excepted thumbnail mimetype to be `image/png` was `%s`", typ)
	}

}

type failingThumbnailer struct{}

func (failingThumbnailer) Command(url.URL) (string, []string) {
	return "failcommand", []string{}
}

func TestMakeThumbnailFailure(t *testing.T) {
	store, ref := storageAndBlobRef(t)
	service := NewService(failingThumbnailer{}, 2*time.Second, 5)
	errMake := service.Generate(ref, store, ioutil.Discard)

	if errMake == nil {
		t.Error("expected to fail.")
	}

}

type sleepyThumbnailer struct{}

func (sleepyThumbnailer) Command(url.URL) (string, []string) {
	return "bash", []string{"-c", `echo "MAY SHOW" 1>&2; sleep 10; echo "SHOULD NEVER SHOW" 1>&2`}
}

func TestThumbnailGenerateTimeout(t *testing.T) {
	service := NewService(sleepyThumbnailer{}, time.Duration(0), 5)
	if err := service.Available(); err != nil {
		t.Skip(err)
	}
	store, ref := storageAndBlobRef(t)
	err := service.Generate(ref, store, ioutil.Discard)

	if err != errTimeout {
		t.Errorf("expected to timeout: %v", err)
	}
}
