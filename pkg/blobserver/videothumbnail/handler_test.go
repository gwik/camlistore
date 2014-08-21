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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"camlistore.org/pkg/blob"
)

func refRequest(path string) *http.Request {
	req, _ := http.NewRequest("GET", path, nil)
	req.RemoteAddr = "[::1]:1234"
	req.Host = "[::1]:5000"
	return req
}

func TestHandlerWrongRef(t *testing.T) {
	store := &blob.MemoryStore{}
	ref, _ := blob.Parse("sha1-f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
	wrongRefString := "sha1-e242ed3bffccdf271b7fbaf34ed72d089537b42f"
	req := refRequest("/" + wrongRefString)
	handler := CreateVideothumbnailHandler(ref, store, 0)

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != 403 {
		t.Fatalf("excepted forbidden status when the wrong ref is requested")
	}

}

func TestHandlerRightRef(t *testing.T) {
	data := "foobarbaz"
	store := &blob.MemoryStore{}
	ref, _ := store.AddBlob(crypto.SHA1, data)
	req := refRequest("/" + ref.String())

	handler := CreateVideothumbnailHandler(ref, store, 0)

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Fatalf("expected 200 status: %v", resp)
	}

	if resp.Body.String() != data {
		t.Errorf("excepted handler to serve data")
	}
}

func TestHandlerRightWithSuffix(t *testing.T) {
	data := "foobarbaz"
	store := &blob.MemoryStore{}
	ref, _ := store.AddBlob(crypto.SHA1, data)
	req := refRequest(fmt.Sprintf("/%s/out.avi", ref.String()))

	handler := CreateVideothumbnailHandler(ref, store, 0)

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Fatalf("expected 200 status: %v", resp)
	}

	if resp.Body.String() != data {
		t.Errorf("excepted handler to serve data")
	}
}
