package main

import (
	"fmt"
	"log"
	"math"
	"runtime"

	"github.com/go-gl/gl"
	"github.com/veandco/go-sdl2/sdl"
	ttf "github.com/veandco/go-sdl2/sdl_ttf"
)

type DrawCommand int

const (
	DRAW_QUIT = iota
	DRAW_FULLSCREEN
)

func initScreen(width, height int) (*sdl.Window, *sdl.Renderer) {
	w, r, err := sdl.CreateWindowAndRenderer(width, height, sdl.WINDOW_OPENGL|sdl.WINDOW_INPUT_GRABBED|sdl.WINDOW_FULLSCREEN)
	if err != nil {
		log.Fatalf(`can't create window: %s`, err)
	}

	if gl.Init() != 0 {
		log.Fatalln(`can't init GL`)
	}

	gl.ClearColor(0.1, 0.1, 0.1, 0.1)

	return w, r
}

func drawSphere(radius float64, lat, lon int) {
	gl.Color3f(1, 1, 1)
	for i := 0; i <= lat; i++ {
		lat0 := math.Pi * (-0.5 + float64(i-1)/float64(lat))
		z0 := math.Sin(lat0)
		zr0 := math.Cos(lat0)

		lat1 := math.Pi * (-0.5 + float64(i)/float64(lat))
		z1 := math.Sin(lat1)
		zr1 := math.Cos(lat1)

		gl.Begin(gl.QUAD_STRIP)
		for j := 0; j <= lon; j++ {
			lng := 2 * math.Pi * (float64(j-1) / float64(lon))
			x := math.Cos(lng)
			y := math.Sin(lng)

			gl.Normal3f(float32(x*zr0), float32(y*zr0), float32(z0))
			gl.Vertex3f(float32(x*zr0), float32(y*zr0), float32(z0))
			gl.Normal3f(float32(x*zr1), float32(y*zr1), float32(z1))
			gl.Vertex3f(float32(x*zr1), float32(y*zr1), float32(z1))
		}
		gl.End()
	}
}

func drawGrid() {
	for i := float32(-500); i <= 500; i += 5 {
		gl.Begin(gl.LINES)
		gl.Color3f(0.2, 0.2, 0.2)
		gl.Vertex3f(-500, i, 0)
		gl.Vertex3f(500, i, 0)
		gl.Vertex3f(i, -500, 0)
		gl.Vertex3f(i, 500, 0)
		gl.End()
	}
}

func createHudSurface(fnt *ttf.Font, fps int64, cam *Camera) *sdl.Surface {
	color := sdl.Color{0, 255, 255, 255}

	lines := []string{
		fmt.Sprintf(`α: %0.2f θ: %0.2f`, cam.alpha, cam.theta),
		fmt.Sprintf(`x: %0.2f y: %0.2f z: %0.2f`, cam.x, cam.y, cam.z),
		fmt.Sprintf(`FPS: %d`, fps),
	}

	w, h := int32(0), int32(0)
	surfaces := []*sdl.Surface{}
	for _, l := range lines {
		s, err := fnt.RenderUTF8_Blended(l, color)
		if err != nil {
			log.Fatalf(`can't render text: %s`, err)
		}
		defer s.Free()
		surfaces = append(surfaces, s)

		if s.W > w {
			w = s.W
		}
		h += s.H
	}

	fmt := surfaces[0].Format

	srf, err := sdl.CreateRGBSurface(0, w, h, 32, fmt.Rmask, fmt.Gmask, fmt.Bmask, fmt.Amask)
	if err != nil {
		log.Fatalf(`can't create SDL surface: %s`, err)
	}
	srf.FillRect(nil, sdl.MapRGBA(srf.Format, 0, 0, 0, 255))

	y := int32(0)
	for _, s := range surfaces {
		s.Blit(nil, srf, &sdl.Rect{Y: y, W: s.W, H: s.H})
		y += s.H
	}

	return srf
}

func drawHud(width, height int, fnt *ttf.Font, r *sdl.Renderer, fps int64, cam *Camera) {
	srf := createHudSurface(fnt, fps, cam)
	defer srf.Free()

	txt, err := r.CreateTextureFromSurface(srf)
	if err != nil {
		log.Fatalf(`can't create texture from text surface: %s`, err)
	}

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0.0, float64(width), float64(height), 0.0, -1.0, 1.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Clear(gl.DEPTH_BUFFER_BIT)

	txt.GL_BindTexture(nil, nil)
	defer txt.GL_UnbindTexture()

	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0, 0)
	gl.Vertex2f(0.0, 0.0)
	gl.TexCoord2f(1, 0)
	gl.Vertex2f(float32(srf.W), 0.0)
	gl.TexCoord2f(1, 1)
	gl.Vertex2f(float32(srf.W), float32(srf.H))
	gl.TexCoord2f(0, 1)
	gl.Vertex2f(0.0, float32(srf.H))
	if err = r.Copy(txt, nil, &sdl.Rect{W: srf.W, H: srf.H}); err != nil {
		log.Fatalf(`can't copy texture: %s`, err)
	}
	gl.End()

	gl.PopMatrix()
}

func drawScreen(width, height int, fnt *ttf.Font, cam *Camera, commands chan DrawCommand) {
	/* SDL wants to run on the 'main thread' */
	runtime.LockOSThread()

	fullscreen := false
	w, r := initScreen(width, height)

	gl.Viewport(0, 0, width, height)
	gl.Hint(gl.PERSPECTIVE_CORRECTION_HINT, gl.NICEST)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, 0)

	target_fps := 24
	ticks_per_frame := int64(1000 / target_fps)
	fps := int64(0)

	for {
		ticks_start := sdl.GetTicks()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		/* Do drawing */
		cam.Update()
		drawGrid()
		drawSphere(1.0, 10, 10)
		drawHud(width, height, fnt, r, fps, cam)
		r.Present()

		select {
		case cmd := <-commands:
			switch cmd {
			case DRAW_QUIT:
				return
			case DRAW_FULLSCREEN:
				if fullscreen {
					w.SetFullscreen(0)
				} else {
					w.SetFullscreen(sdl.WINDOW_FULLSCREEN)
				}
				fullscreen = !fullscreen
			}
		default:
			/* ignore */
		}

		tickdelta := int64(sdl.GetTicks()) - int64(ticks_start)
		if tickdelta < 0 {
			tickdelta = 1
		}
		tickdelta = ticks_per_frame - tickdelta

		fps = 1000 / tickdelta

		sdl.Delay(uint32(tickdelta))
	}
}
