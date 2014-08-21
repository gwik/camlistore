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
	"crypto"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/magic"
)

func Store(t *testing.T) (blob.Fetcher, blob.Ref) {
	inFile, err := os.Open("testdata/small.ogv")
	if err != nil {
		t.Fatal(err)
	}
	store := &blob.MemoryStore{}
	data, errRead := ioutil.ReadAll(inFile)
	if errRead != nil {
		t.Fatal(errRead)
	}
	ref, errAdd := store.AddBlob(crypto.SHA1, string(data))
	if errAdd != nil {
		t.Fatal(err)
	}
	return store, ref
}

func TestMakeThumbnail(t *testing.T) {
	store, ref := Store(t)

	tmpFile, _ := ioutil.TempFile(os.TempDir(), "camlitest")
	defer tmpFile.Close()
	service := ThumbnailService{DefaultThumbnailer, 10 * time.Second}
	errMake := service.Generate(ref, store, tmpFile)

	if errMake != nil {
		t.Fatal(errMake)
	}

	tmpFile.Seek(0, 0)

	typ, _ := magic.MIMETypeFromReader(tmpFile)
	if typ != "image/jpeg" {
		t.Errorf("excepted thumbnail mimetype to be `image/jpeg` was `%s`", typ)
	}

}

type failingThumbnailer struct{}

func (_ failingThumbnailer) Command(_ url.URL) (string, []string) {
	return "test", []string{"1", "-ne", "1"}
}

func TestMakeThumbnailFailure(t *testing.T) {
	store, ref := Store(t)

	tmpFile, _ := ioutil.TempFile(os.TempDir(), "camlitest")
	defer tmpFile.Close()
	service := ThumbnailService{failingThumbnailer{}, 10 * time.Second}
	errMake := service.Generate(ref, store, tmpFile)

	if errMake == nil {
		t.Fatal("expected to fail.")
	}

}
