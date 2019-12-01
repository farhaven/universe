package text

import (
	"image/png"
	"image/color"
	"os"
	"testing"
)

func TestNewTextContext(t *testing.T) {
	_, err := NewContext("../../font.ttf")
	if err != nil {
		t.Errorf(`%s`, err)
	}
}

func TestRender(t *testing.T) {
	c, err := NewContext("../../font.ttf")
	if err != nil {
		t.Errorf(`%s`, err)
	}

	gray := color.Gray{128}
		
	img, err := c.Render("π ⚠ fnord", 20, gray)
	if err != nil {
		t.Errorf(`%s`, err)
	}

	w, err := os.Create("test.png")
	if err != nil {
		t.Errorf(`%s`, err)
	}
	defer w.Close()

	if err := png.Encode(w, img); err != nil {
		t.Errorf(`can't dump image: %s`, err)
	}
}

func TestRenderMultiline(t *testing.T) {
	c, err := NewContext("../../font.ttf")
	if err != nil {
		t.Errorf(`%s`, err)
	}

	bg := color.Gray{0}
	fg := color.Gray{127}
	img, err := c.RenderMultiline([]string{"Foo", "Bar", "☺ ⚠"}, 20, bg, fg)
	if err != nil {
		t.Errorf(`%s`, err)
	}

	w, err := os.Create("test2.png")
	if err != nil {
		t.Errorf(`%s`, err)
	}
	defer w.Close()

	if err := png.Encode(w, img); err != nil {
		t.Errorf(`can't dump image: %s`, err)
	}
}
