package services

import (
	"errors"
	"image"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
)

const (
	// CaptchaTypeImage is the classic text captcha (base64Captcha).
	CaptchaTypeImage = "image"
	// CaptchaTypeSlide is the slide-to-fit puzzle captcha (go-captcha).
	CaptchaTypeSlide = "slide"

	// slideStorePrefix marks slide answers in the shared captcha store, so an
	// image-mode verify can never accept a slide token and vice versa.
	slideStorePrefix = "slide:"

	// Puzzle background size in pixels. The API returns it so SPAs render at exactly this size
	// (the slide answer is an x offset in these coordinates).
	slideImageWidth  = 300
	slideImageHeight = 150
	// Puzzle piece size range in pixels.
	slideTileMin = 50
	slideTileMax = 60

	// slideTolerancePx is the accepted horizontal error (pixels) for a slide answer.
	slideTolerancePx = 5
)

// NormalizeCaptchaType maps any config value to a supported captcha type (default image).
func NormalizeCaptchaType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case CaptchaTypeSlide:
		return CaptchaTypeSlide
	default:
		return CaptchaTypeImage
	}
}

// CaptchaChallenge is the payload issued to a client for one captcha attempt.
type CaptchaChallenge struct {
	Type string
	ID   string
	// Image is the data URL of the text captcha (image mode).
	Image string
	// Slide mode fields.
	MasterImage string
	TileImage   string
	MasterWidth  int
	MasterHeight int
	TileWidth    int
	TileHeight   int
	TileX        int
	TileY        int
}

var (
	slideMu      sync.Mutex
	slideCaptcha slide.Captcha
)

// getSlideCaptcha lazily builds the slide generator (decodes bundled images once).
func getSlideCaptcha() (slide.Captcha, error) {
	slideMu.Lock()
	defer slideMu.Unlock()
	if slideCaptcha != nil {
		return slideCaptcha, nil
	}

	backgrounds, err := imagesv2.GetImages()
	if err != nil {
		return nil, err
	}
	if len(backgrounds) == 0 {
		return nil, errors.New("slide captcha: no background images")
	}
	graphs, err := tiles.GetTiles()
	if err != nil {
		return nil, err
	}
	graphImages := make([]*slide.GraphImage, 0, len(graphs))
	for _, g := range graphs {
		graphImages = append(graphImages, &slide.GraphImage{
			OverlayImage: g.OverlayImage,
			ShadowImage:  g.ShadowImage,
			MaskImage:    g.MaskImage,
		})
	}

	builder := slide.NewBuilder()
	builder.SetOptions(
		slide.WithImageSize(option.Size{Width: slideImageWidth, Height: slideImageHeight}),
		slide.WithRangeGraphSize(option.RangeVal{Min: slideTileMin, Max: slideTileMax}),
	)
	builder.SetResources(
		slide.WithBackgrounds(append([]image.Image(nil), backgrounds...)),
		slide.WithGraphImages(graphImages),
	)
	slideCaptcha = builder.Make()
	return slideCaptcha, nil
}

// generateSlide renders a slide puzzle and returns the challenge plus the correct x offset.
func generateSlide() (*CaptchaChallenge, int, error) {
	capt, err := getSlideCaptcha()
	if err != nil {
		return nil, 0, err
	}

	// Generation is serialized: the generator shares option/resource state.
	slideMu.Lock()
	data, err := capt.Generate()
	slideMu.Unlock()
	if err != nil {
		return nil, 0, err
	}

	block := data.GetData()
	if block == nil {
		return nil, 0, errors.New("slide captcha: empty block")
	}
	master, err := data.GetMasterImage().ToBase64()
	if err != nil {
		return nil, 0, err
	}
	tile, err := data.GetTileImage().ToBase64()
	if err != nil {
		return nil, 0, err
	}

	return &CaptchaChallenge{
		Type:        CaptchaTypeSlide,
		MasterImage: master,
		TileImage:   tile,
		MasterWidth:  slideImageWidth,
		MasterHeight: slideImageHeight,
		TileWidth:    block.Width,
		TileHeight:   block.Height,
		TileX:        block.DX,
		TileY:        block.DY,
	}, block.X, nil
}

// verifySlideAnswer checks a client answer ("x" or "x,y") against the stored x offset.
func verifySlideAnswer(expectedX int, answer string) bool {
	part := strings.TrimSpace(answer)
	if i := strings.Index(part, ","); i >= 0 {
		part = strings.TrimSpace(part[:i])
	}
	got, err := strconv.ParseFloat(part, 64)
	if err != nil || math.IsNaN(got) || math.IsInf(got, 0) {
		return false
	}
	return math.Abs(got-float64(expectedX)) <= slideTolerancePx
}

// ToMap renders the challenge as the public API payload (shared by admin and platform login).
func (ch *CaptchaChallenge) ToMap() map[string]any {
	out := map[string]any{
		"type":       ch.Type,
		"captcha_id": ch.ID,
	}
	if ch.Type == CaptchaTypeSlide {
		out["master_image"] = ch.MasterImage
		out["tile_image"] = ch.TileImage
		out["master_width"] = ch.MasterWidth
		out["master_height"] = ch.MasterHeight
		out["tile_width"] = ch.TileWidth
		out["tile_height"] = ch.TileHeight
		out["tile_x"] = ch.TileX
		out["tile_y"] = ch.TileY
	} else {
		out["captcha_image"] = ch.Image
	}
	return out
}
