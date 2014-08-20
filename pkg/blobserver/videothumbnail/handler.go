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
	"net/http"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver/gethandler"
)

type Handler struct {
	ref     blob.Ref
	fetcher blob.Fetcher
}

func (handler Handler) ServeHTTP(conn http.ResponseWriter, req *http.Request) {

	// TODO: verify auth with ident

	blobRef, ok := blob.Parse(req.URL.Path)
	if !ok || !blobRef.Valid() {
		http.Error(conn, "Malformed GET URL.", 400)
		return
	}

	// handler only serves its `ref`
	if blobRef != handler.ref {
		http.Error(conn, "Forbidden", 403)
		return
	}

	gethandler.ServeBlobRef(conn, req, handler.ref, handler.fetcher)
}

func CreateVideothumbnailHandler(ref blob.Ref, fetcher blob.Fetcher) http.Handler {
	handler := Handler{ref: ref, fetcher: fetcher}
	return handler
}
