package demo

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	imgpkg "github.com/gammons/slk/internal/image"
)

func TestChartAttachmentIsAnInlineImage(t *testing.T) {
	a := chartAttachment()
	if a.Kind != "image" || a.FileID != chartFileID {
		t.Fatalf("attachment = %+v, want an image with FileID %q", a, chartFileID)
	}
	if len(a.Thumbs) != 1 || a.Thumbs[0].W != chartW || a.Thumbs[0].H != chartH || a.Thumbs[0].URL == "" {
		t.Fatalf("thumbs = %+v, want one %dx%d thumb with a URL", a.Thumbs, chartW, chartH)
	}
}

func TestDrawChartHasBarsOnABackground(t *testing.T) {
	img := drawChart()
	if img.Bounds() != image.Rect(0, 0, chartW, chartH) {
		t.Fatalf("bounds = %v", img.Bounds())
	}
	bg := img.RGBAAt(1, 1)
	slot := chartW / 17
	lastBar := img.RGBAAt(slot*15+slot/2, chartH-30)
	if lastBar == bg {
		t.Fatal("the last week's bar is not drawn")
	}
	if want := (color.RGBA{0x9e, 0xce, 0x6a, 0xff}); lastBar != want {
		t.Errorf("last bar colour = %v, want highlight %v", lastBar, want)
	}
}

func TestChartFetcherServesOnlyTheChart(t *testing.T) {
	f := chartFetcher{img: drawChart()}
	if img, ok := f.Cached(chartFileID+"-640", image.Pt(320, 160)); !ok || img == nil {
		t.Fatal("chart key is not cached")
	}
	if _, ok := f.Cached("F0OTHER-360", image.Pt(320, 160)); ok {
		t.Fatal("unknown key reported as cached")
	}
	if _, err := f.Fetch(context.Background(), imgpkg.FetchRequest{Key: "F0OTHER-360"}); !errors.Is(err, errUnavailable) {
		t.Fatalf("Fetch(unknown) err = %v, want errUnavailable", err)
	}
	res, err := f.Fetch(context.Background(), imgpkg.FetchRequest{Key: chartFileID + "-640"})
	if err != nil || res.Img == nil {
		t.Fatalf("Fetch(chart) = %+v, %v", res, err)
	}
	if _, ok := f.Prerendered(chartFileID+"-640", image.Pt(40, 10), imgpkg.ProtoHalfBlock); ok {
		t.Fatal("Prerendered must miss so the renderer encodes half-blocks itself")
	}
}
