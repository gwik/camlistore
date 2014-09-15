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

package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/constants"
	"camlistore.org/pkg/images"
	"camlistore.org/pkg/schema"
	"camlistore.org/pkg/syncutil"
	"camlistore.org/pkg/test"
	"camlistore.org/pkg/videothumbnail"
)

const videoFilepath = "../videothumbnail/testdata/small.webm"
const imageFilepath = "../images/testdata/f1.jpg"

func newTestImageHandler(storage blob.Fetcher, maxWidth, maxHeight int, cached bool) *ImageHandler {
	var (
		cache blobserver.Storage
		meta  *ThumbMeta
	)

	if cached {
		cache = new(test.Fetcher)
		meta = NewThumbMeta(nil)
	}

	return &ImageHandler{
		Fetcher:   storage,
		Cache:     cache,
		MaxWidth:  maxWidth,
		MaxHeight: maxHeight,
		ThumbMeta: meta,
		ResizeSem: syncutil.NewSem(constants.DefaultMaxResizeMem),
		VideoThumbnail: videothumbnail.NewService(
			videothumbnail.DefaultThumbnailer, time.Duration(5)*time.Second, 5),
	}
}

func addFile(t *testing.T, storage blobserver.Storage, filename string) blob.Ref {
	inFile, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer inFile.Close()
	ref, err := schema.WriteFileFromReader(storage, path.Base(filename), inFile)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestImageThumbnailNoCaches(t *testing.T) {
	storage := new(test.Fetcher)
	// a 40x80 image
	ref := addFile(t, storage, imageFilepath)
	maxWidth := 40
	maxHeight := 40
	path := fmt.Sprintf(
		"thumbnail/%s/%s?mh=%d&tv=%s&v=%t",
		ref, path.Base(imageFilepath), maxHeight, images.ThumbnailVersion(), false)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := httptest.NewRecorder()
	handler := newTestImageHandler(storage, maxWidth, maxHeight, false)

	handler.ServeHTTP(resp, req, ref)
	if resp.Code != 200 {
		t.Fatal("expected 200 status code.")
	}
	header := resp.Header()
	if ct := header.Get("Content-Type"); !strings.HasPrefix(ct, "image/jpeg") {
		t.Errorf("expected `image/jpeg` content type, was: `%s`", ct)
	}

	config, err := images.DecodeConfig(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if config.Format != "jpeg" {
		t.Errorf("expected `jpeg` format, was: `%s`", config.Format)
	}
	if config.Height != maxHeight {
		t.Errorf("expected to be resized to max height was: `%d`", config.Height)
	}
}

func TestVideoThumbnailNoCaches(t *testing.T) {
	storage := new(test.Fetcher)
	ref := addFile(t, storage, videoFilepath)
	maxWidth := 40
	maxHeight := 40
	path := fmt.Sprintf(
		"thumbnail/%s/%s?mh=%d&tv=%s&v=%t",
		ref, path.Base(imageFilepath), maxHeight, images.ThumbnailVersion(), true)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := httptest.NewRecorder()
	handler := newTestImageHandler(storage, maxWidth, maxHeight, false)

	handler.ServeHTTP(resp, req, ref)
	if resp.Code != 200 {
		t.Fatal("expected 200 status code.")
	}
	header := resp.Header()
	if ct := header.Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Errorf("expected `image/png` content type, was: `%s`", ct)
	}

	config, err := images.DecodeConfig(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if config.Format != "png" {
		t.Errorf("expected `png` format, was: `%s`", config.Format)
	}
	if config.Height > maxHeight {
		t.Errorf("expected to be resized to max height(%d) was: %d",
			maxHeight, config.Height)
	}
	if config.Height > maxWidth {
		t.Errorf("expected to be resized to max width(%d) was: %d",
			maxWidth, config.Width)
	}
}

func TestVideoThumbnailCached(t *testing.T) {
	storage := new(test.Fetcher)
	ref := addFile(t, storage, videoFilepath)
	maxWidth := 40
	maxHeight := 40
	path := fmt.Sprintf(
		"thumbnail/%s/%s?mh=%d&tv=%s&v=%t",
		ref, path.Base(imageFilepath), maxHeight, images.ThumbnailVersion(), true)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := httptest.NewRecorder()
	handler := newTestImageHandler(storage, maxWidth, maxHeight, true)
	handler.ServeHTTP(resp, req, ref)
	if resp.Code != 200 {
		t.Fatal("Expected 200 status code.")
	}

	originalThumbKey := cacheKey(ref.String(), 0, 0)
	format := handler.scaledCached(&bytes.Buffer{}, originalThumbKey)
	if format == "" {
		t.Error("Expected original thumbnail file to be in the cache.")
	}

	thumbKey := cacheKey(ref.String(), maxWidth, maxHeight)
	format = handler.scaledCached(&bytes.Buffer{}, thumbKey)
	if format == "" {
		t.Error("Expected thumbnail file to be in the cache.")
	}
}
