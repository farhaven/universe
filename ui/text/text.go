package text

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"os"

	"github.com/golang/freetype"
	"golang.org/x/image/math/fixed"
)

type Context struct {
	ft *freetype.Context
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
	ctx.SetDPI(72)

	return &Context{ctx}, nil
}

func int26_6ToFloat64(i fixed.Int26_6) float64 {
	return float64(i>>6) + 1/(float64(i&((1<<7)-1))+1)
}

type infiniteImage struct{}

func (i infiniteImage) ColorModel() color.Model {
	return color.RGBAModel
}
func (i infiniteImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, 1, 1)
}
func (i infiniteImage) At(x, y int) color.Color {
	return color.Black
}
func (i infiniteImage) Set(x, y int, c color.Color) {
}

func (c *Context) Render(txt string, size float64, col color.Color) (*image.RGBA, error) {
	c.ft.SetSrc(image.NewUniform(col))
	c.ft.SetFontSize(size)

	/* Render image to temporary buffer to determine final size */
	tmp := infiniteImage{}
	c.ft.SetDst(tmp)
	c.ft.SetClip(tmp.Bounds())
	p, err := c.ft.DrawString(txt, fixed.P(0, int(size+0.5)))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, int(int26_6ToFloat64(p.X)+0.5), int(int26_6ToFloat64(p.Y)+0.5)))
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
