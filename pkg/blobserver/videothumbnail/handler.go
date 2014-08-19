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

	// handler only serves `ref`
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
