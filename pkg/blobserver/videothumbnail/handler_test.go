package videothumbnail

import (
	"crypto"
	"net/http"
	"net/http/httptest"
	"testing"

	"camlistore.org/pkg/blob"
)

func TestHandlerWrongRef(t *testing.T) {
	store := &blob.MemoryStore{}
	ref, _ := blob.Parse("sha1-f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
	wrongRefString := "sha1-e242ed3bffccdf271b7fbaf34ed72d089537b42f"
	req, _ := http.NewRequest("GET", wrongRefString, nil)

	handler := CreateVideothumbnailHandler(ref, store)

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != 403 {
		t.Errorf("excepted forbidden status when the wrong ref is requested")
	}

}

func TestHandlerRightRef(t *testing.T) {
	data := "foobarbaz"
	store := &blob.MemoryStore{}
	ref, _ := store.AddBlob(crypto.SHA1, data)
	req, _ := http.NewRequest("GET", ref.String(), nil)

	handler := CreateVideothumbnailHandler(ref, store)

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("excepted forbidden status when the wrong ref is requested")
	}

	if resp.Body.String() != data {
		t.Errorf("excepted handler to serve data")
	}
}
