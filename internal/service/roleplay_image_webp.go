package service

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"

	"github.com/chai2010/webp"
)

const rolePlayImageWebPMime = "image/webp"

func rolePlayImageWebPQuality() int {
	quality := envInt("ROLEPLAY_IMAGE_WEBP_QUALITY", 82)
	if quality < 50 {
		return 50
	}
	if quality > 95 {
		return 95
	}
	return quality
}

func rolePlayImageMaxWidth() int {
	maxWidth := envInt("ROLEPLAY_IMAGE_MAX_WIDTH", 1600)
	if maxWidth < 0 {
		return 0
	}
	return maxWidth
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := parseInt64(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return int(parsed)
}

func rolePlayAllowedUploadExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

type rolePlayWebPImage struct {
	Data   []byte
	Width  int
	Height int
}

func decodeRolePlayUploadImage(data []byte, mimeType string) (image.Image, error) {
	if strings.EqualFold(strings.TrimSpace(mimeType), rolePlayImageWebPMime) {
		return webp.Decode(bytes.NewReader(data))
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func resizeRolePlayImageToMaxWidth(src image.Image, maxWidth int) image.Image {
	if maxWidth <= 0 {
		return src
	}
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= maxWidth {
		return src
	}
	dstW := maxWidth
	dstH := int(float64(srcH) * float64(dstW) / float64(srcW))
	if dstH <= 0 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		srcY := bounds.Min.Y + y*srcH/dstH
		for x := 0; x < dstW; x++ {
			srcX := bounds.Min.X + x*srcW/dstW
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// ConvertUploadedImageBytesToWebP validates input bytes and returns normalized WebP output.
func ConvertUploadedImageBytesToWebP(data []byte, originalName string) (*rolePlayWebPImage, string, string, error) {
	if len(data) == 0 {
		return nil, "", "", fmt.Errorf("empty image file")
	}
	ext, originalMime, err := validateRolePlayImageUpload(originalName, data)
	if err != nil {
		return nil, "", "", err
	}

	img, err := decodeRolePlayUploadImage(data, originalMime)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid or corrupted image")
	}

	img = resizeRolePlayImageToMaxWidth(img, rolePlayImageMaxWidth())
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var webpBuf bytes.Buffer
	if err := webp.Encode(&webpBuf, img, &webp.Options{Quality: float32(rolePlayImageWebPQuality())}); err != nil {
		return nil, "", "", fmt.Errorf("webp conversion failed")
	}
	if webpBuf.Len() == 0 {
		return nil, "", "", fmt.Errorf("webp conversion produced empty output")
	}

	return &rolePlayWebPImage{
		Data:   webpBuf.Bytes(),
		Width:  width,
		Height: height,
	}, ext, originalMime, nil
}

// ConvertUploadedImageToWebP converts a file on disk to WebP and removes the source file on success.
func ConvertUploadedImageToWebP(inputPath string, outputPath string, quality int) error {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	converted, _, _, err := ConvertUploadedImageBytesToWebP(input, inputPath)
	if err != nil {
		return err
	}
	if quality > 0 && quality != rolePlayImageWebPQuality() {
		img, decodeErr := webp.Decode(bytes.NewReader(converted.Data))
		if decodeErr != nil {
			return decodeErr
		}
		var webpBuf bytes.Buffer
		if encodeErr := webp.Encode(&webpBuf, img, &webp.Options{Quality: float32(quality)}); encodeErr != nil {
			return encodeErr
		}
		converted.Data = webpBuf.Bytes()
	}
	if err := os.WriteFile(outputPath, converted.Data, 0o644); err != nil {
		return err
	}
	return os.Remove(inputPath)
}

func convertReaderToWebP(reader io.Reader, originalName string) (*rolePlayWebPImage, string, string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", "", err
	}
	return ConvertUploadedImageBytesToWebP(data, originalName)
}