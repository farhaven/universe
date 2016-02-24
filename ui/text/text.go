package text

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

type Context struct {
	ft *freetype.Context
	fnt *truetype.Font
}

func NewContext(font string) (*Context, error) {
	fh, err := os.Open(font)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	data, err := ioutil.ReadAll(fh)
	if err != nil {
		return nil, err
	}

	fnt, err := freetype.ParseFont(data)
	if err != nil {
		return nil, err
	}

	ctx := freetype.NewContext()
	ctx.SetFont(fnt)
	/* XXX: get appropriate DPI for current display */
	ctx.SetDPI(96)

	return &Context{ctx, fnt}, nil
}

func int26_6ToFloat64(i fixed.Int26_6) float64 {
	return float64(i>>6) + 1/(float64(i&((1<<7)-1))+1)
}

type nullImage struct{}

func (i nullImage) ColorModel() color.Model {
	return color.RGBAModel
}
func (i nullImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, 1, 1)
}
func (i nullImage) At(x, y int) color.Color {
	return color.Black
}
func (i nullImage) Set(x, y int, c color.Color) {
}

func (c *Context) Render(txt string, size float64, col color.Color) (*image.RGBA, error) {
	bnd := c.fnt.Bounds(fixed.I(int(size + 0.5)))
	lh := int26_6ToFloat64(bnd.Max.Y) - int26_6ToFloat64(bnd.Min.Y) - 0.5

	c.ft.SetSrc(image.NewUniform(col))
	c.ft.SetFontSize(size)

	/* Render image to temporary buffer to determine final size */
	tmp := nullImage{}
	c.ft.SetDst(tmp)
	c.ft.SetClip(tmp.Bounds())
	p, err := c.ft.DrawString(txt, fixed.P(0, int(lh)))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, int(int26_6ToFloat64(p.X)+0.5), int(lh)))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{}), image.ZP, draw.Src)
	c.ft.SetDst(dst)
	c.ft.SetClip(dst.Bounds())

	p, err = c.ft.DrawString(txt, fixed.P(0, int(size)))
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func (c *Context) RenderMultiline(txt []string, size float64, bg, fg color.Color) (*image.RGBA, error) {
	w, h := 0, 0
	imgs := []*image.RGBA{}

	for _, l := range txt {
		i, err := c.Render(l, size, fg)
		if err != nil {
			return nil, err
		}
		if i.Bounds().Dx() > w {
			w = i.Bounds().Dx()
		}
		h += i.Bounds().Dy()
		imgs = append(imgs, i)
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(bg), image.ZP, draw.Src)
	y := 0
	for _, src := range imgs {
		sr := src.Bounds()
		dp := image.Point{0, y}
		r := image.Rectangle{dp, dp.Add(sr.Size())}
		draw.Draw(dst, r, src, sr.Min, draw.Src)
		y += sr.Dy()
	}

	return dst, nil
}
