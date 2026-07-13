package service

import (
	"mime/multipart"
	"reflect"
	"testing"
)

func TestParseGrokMediaMultipartFormUsesDeterministicAliasOrder(t *testing.T) {
	form := &multipart.Form{Value: map[string][]string{
		"image":          {"legacy-image"},
		"image_url":      {"current-image"},
		"mask":           {"legacy-mask"},
		"mask_image_url": {"current-mask"},
	}}

	for range 100 {
		info := ParseGrokMediaMultipartForm(form)
		if !reflect.DeepEqual(info.InputImageURLs, []string{"legacy-image", "current-image"}) {
			t.Fatalf("input images = %#v", info.InputImageURLs)
		}
		if info.MaskImageURL != "current-mask" {
			t.Fatalf("mask = %q, want current-mask", info.MaskImageURL)
		}
	}
}
