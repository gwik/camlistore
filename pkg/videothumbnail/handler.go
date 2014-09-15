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
	"log"
	"net/http"
	"strings"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/httputil"
	"camlistore.org/pkg/schema"
)

const (
	contentTypeOctetStream = "application/octet-stream"
	httpHeaderContentType  = "Content-Type"
	httpMethodGet          = "GET"
	httpMethodHead         = "HEAD"
)

type handler struct {
	ref     blob.Ref
	fetcher blob.Fetcher
}

func (h *handler) auth(req *http.Request) bool {
	// check pid of the process ??
	return httputil.IsLocalhost(req)
}

func (h *handler) ServeHTTP(conn http.ResponseWriter, req *http.Request) {
	if req.Method != httpMethodGet && req.Method != httpMethodHead {
		http.Error(conn, "Invalid download method", 400)
		return
	}

	if !h.auth(req) {
		http.Error(conn, "Forbidden.", 403)
		return
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(conn, "Malformed GET URL.", 400)
		return
	}
	ref := parts[1]

	blobRef, ok := blob.Parse(ref)
	if !ok || !blobRef.Valid() {
		http.Error(conn, "Malformed GET URL.", 400)
		return
	}

	// h only serves its ref
	if blobRef != h.ref {
		log.Println("forbidden access to blobRef " + blobRef.String())
		http.Error(conn, "Forbidden.", 403)
		return
	}

	header := conn.Header()
	header.Set(httpHeaderContentType, contentTypeOctetStream)

	fr, err := schema.NewFileReader(h.fetcher, h.ref)
	if err != nil {
		httputil.ServeError(conn, req, err)
		return
	}
	defer fr.Close()

	http.ServeContent(conn, req, "", time.Now(), fr)
}

func createVideothumbnailHandler(ref blob.Ref, fetcher blob.Fetcher) http.Handler {
	return &handler{ref: ref, fetcher: fetcher}
}
