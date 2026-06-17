package service

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestConvertUploadedImageBytesToWebPFromPNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 4))
	var input bytes.Buffer
	if err := png.Encode(&input, src); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	converted, _, originalMime, err := ConvertUploadedImageBytesToWebP(input.Bytes(), "scene.png")
	if err != nil {
		t.Fatalf("convert png: %v", err)
	}
	if originalMime != "image/png" {
		t.Fatalf("expected original mime image/png, got %s", originalMime)
	}
	if len(converted.Data) == 0 {
		t.Fatal("expected webp bytes")
	}
	if !strings.HasPrefix(string(converted.Data), "RIFF") {
		t.Fatal("expected webp RIFF header")
	}
	if converted.Width != 8 || converted.Height != 4 {
		t.Fatalf("unexpected dimensions %dx%d", converted.Width, converted.Height)
	}
}

func TestRolePlayImageWebPQualityBounds(t *testing.T) {
	t.Setenv("ROLEPLAY_IMAGE_WEBP_QUALITY", "10")
	if got := rolePlayImageWebPQuality(); got != 50 {
		t.Fatalf("expected min quality 50, got %d", got)
	}
	t.Setenv("ROLEPLAY_IMAGE_WEBP_QUALITY", "120")
	if got := rolePlayImageWebPQuality(); got != 95 {
		t.Fatalf("expected max quality 95, got %d", got)
	}
}