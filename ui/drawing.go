package ui

import (
	"fmt"
	"log"
	"math"
	"runtime"

	"../orrery"
	"../vector"

	"github.com/go-gl-legacy/gl"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/veandco/go-sdl2/sdl"
	ttf "github.com/veandco/go-sdl2/sdl_ttf"
)

type DrawCommand int

const (
	DRAW_QUIT = iota
	DRAW_FULLSCREEN
	DRAW_TOGGLE_WIREFRAME
)

type DrawContext struct {
	width, height int
	win           *sdl.Window
	renderer      *sdl.Renderer
	cmd           chan DrawCommand

	cam *Camera

	fnt       *ttf.Font
	wireframe bool

	shutdown chan struct{}
}

func initSDL() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Fatalf(`SDL Init failed: %s`, err)
	}
	if err := sdl.VideoInit(sdl.GetVideoDriver(0)); err != nil {
		log.Fatalf(`can't init video: %s`, err)
	}
}

func loadFont() *ttf.Font {
	if err := ttf.Init(); err != nil {
		log.Fatalf(`can't init font system: %s`, err)
	}

	fnt, err := ttf.OpenFont("font.ttf", 12)
	if err != nil {
		log.Fatalf(`can't load font.ttf: %s`, err)
	}

	return fnt
}

func NewDrawContext(width, height int, o *orrery.Orrery) DrawContext {
	initSDL()

	fnt := loadFont()

	cam := NewCamera(width, height, -40, 40, 10)

	c := make(chan DrawContext)

	// This is a hack to make sure all drawing stuff runs in the same goroutine
	// XXX: This should probably be replaced by a goroutine that listens to a channel for GL commands
	go func() {
		/* SDL wants to run on the 'main thread' */
		runtime.LockOSThread()

		w, r, err := sdl.CreateWindowAndRenderer(width, height, sdl.WINDOW_OPENGL|sdl.WINDOW_INPUT_GRABBED)
		if err != nil {
			log.Fatalf(`can't create window: %s`, err)
		}

		if gl.Init() != 0 {
			log.Fatalln(`can't init GL`)
		}

		gl.ClearColor(0.1, 0.1, 0.1, 0.1)
		gl.Enable(gl.DEPTH_TEST)

		ctx := DrawContext{
			width: width, height: height,
			win: w, renderer: r,
			cmd:       make(chan DrawCommand),
			wireframe: true,
			cam:       cam,
			fnt:       fnt,
			shutdown:  make(chan struct{}),
		}
		c <- ctx

		go ctx.EventLoop(o)
		ctx.drawScreen(o)
		close(ctx.shutdown)
	}()

	return <-c
}

func (ctx *DrawContext) WaitForShutdown() {
	<-ctx.shutdown
}

func (ctx *DrawContext) Shutdown() {
	sdl.VideoQuit()
	sdl.Quit()
}

func (ctx *DrawContext) QueueCommand(cmd DrawCommand) {
	ctx.cmd <- cmd
}

func (ctx *DrawContext) drawPlanets(o *orrery.Orrery) {
	for _, p := range o.Planets() {
		ctx.drawPlanet(p)
	}
}

func (ctx *DrawContext) drawPlanet(p *orrery.Planet) {
	c := colorful.Hcl(math.Remainder((math.Pi/p.M)*360, 360), 0.9, 0.9)

	ctx.drawSphere(p.Pos, p.R, c)
	for _, pos := range p.Trail {
		ctx.drawSphere(pos, 1, c)
	}
}

func (ctx *DrawContext) drawSphere(p vector.V3, r float64, c colorful.Color) {
	if ctx.cam.SphereInFrustum(p, r) == OUTSIDE {
		return
	}

	gl.Color3f(float32(c.R), float32(c.G), float32(c.B))

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	defer gl.PopMatrix()

	slices := int(math.Max(10, 5*math.Log(r+1)))

	gl.Translated(p.X, p.Y, p.Z)
	gl.Scaled(r, r, r)

	for i := 0; i <= slices; i++ {
		lat0 := math.Pi * (-0.5 + float64(i-1)/float64(slices))
		z0 := math.Sin(lat0)
		zr0 := math.Cos(lat0)

		lat1 := math.Pi * (-0.5 + float64(i)/float64(slices))
		z1 := math.Sin(lat1)
		zr1 := math.Cos(lat1)

		if ctx.wireframe {
			gl.Begin(gl.LINES)
		} else {
			gl.Begin(gl.QUAD_STRIP)
		}
		for j := 0; j <= slices; j++ {
			lng := 2 * math.Pi * (float64(j-1) / float64(slices))
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

func (ctx *DrawContext) drawGrid() {
	gl.Disable(gl.DEPTH_TEST)
	defer gl.Enable(gl.DEPTH_TEST)

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

func (ctx *DrawContext) createHudSurface(o *orrery.Orrery, tpf int64) *sdl.Surface {
	color := sdl.Color{0, 255, 255, 255}

	lines := []string{
		"WASD: Move, 1: Toggle wireframe, F: Fullscreen, Q: Quit",
		"Mouse Wheel: Move fast, Mouse Btn #1: Spawn planet",
		"P: panic and dump stacks",
		fmt.Sprintf(` α: %0.2f θ: %0.2f`, ctx.cam.alpha, ctx.cam.theta),
		fmt.Sprintf(` x: %0.2f y: %0.2f z: %0.2f`, ctx.cam.Pos.X, ctx.cam.Pos.Y, ctx.cam.Pos.Z),
		fmt.Sprintf(` Ticks/Frame: %d`, tpf),
	}

	for i, p := range o.Planets() {
		l := fmt.Sprintf(` π %d: r=%0.2f M=%0.2f pos=(%0.2f, %0.2f, %0.2f), vel=(%0.2f, %0.2f, %0.2f) f:%s`, i, p.R, p.M, p.Pos.X, p.Pos.Y, p.Pos.Z, p.Vel.X, p.Vel.Y, p.Vel.Z, ctx.cam.SphereInFrustum(p.Pos, p.R).String())
		lines = append(lines, l)
	}

	w, h := int32(0), int32(0)
	surfaces := []*sdl.Surface{}
	for _, l := range lines {
		s, err := ctx.fnt.RenderUTF8_Blended(l, color)
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

func (ctx *DrawContext) drawHud(o *orrery.Orrery, tpf int64) {
	srf := ctx.createHudSurface(o, tpf)
	defer srf.Free()

	txt, err := ctx.renderer.CreateTextureFromSurface(srf)
	if err != nil {
		log.Fatalf(`can't create texture from text surface: %s`, err)
	}

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0.0, float64(ctx.width), float64(ctx.height), 0.0, -1.0, 1.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Clear(gl.DEPTH_BUFFER_BIT)

	txt.GL_BindTexture(nil, nil)
	defer txt.GL_UnbindTexture()

	gl.Color3f(1, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0, 0)
	gl.Vertex2f(0.0, 0.0)
	gl.TexCoord2f(1, 0)
	gl.Vertex2f(float32(srf.W), 0.0)
	gl.TexCoord2f(1, 1)
	gl.Vertex2f(float32(srf.W), float32(srf.H))
	gl.TexCoord2f(0, 1)
	gl.Vertex2f(0.0, float32(srf.H))
	if err = ctx.renderer.Copy(txt, nil, &sdl.Rect{W: srf.W, H: srf.H}); err != nil {
		log.Fatalf(`can't copy texture: %s`, err)
	}
	gl.End()

	gl.PopMatrix()
}

func (ctx *DrawContext) drawScreen(o *orrery.Orrery) {
	fullscreen := false

	gl.Viewport(0, 0, ctx.width, ctx.height)
	gl.Hint(gl.PERSPECTIVE_CORRECTION_HINT, gl.NICEST)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, 0)

	target_tpf := 24
	ticks_per_frame := int64(1000 / target_tpf)
	tpf := int64(0)

	for {
		ticks_start := sdl.GetTicks()
		o.Step()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		ctx.cam.Update()
		ctx.drawGrid()
		ctx.drawPlanets(o)
		ctx.drawHud(o, tpf)
		ctx.renderer.Present()

		select {
		case cmd := <-ctx.cmd:
			switch cmd {
			case DRAW_QUIT:
				return
			case DRAW_FULLSCREEN:
				if fullscreen {
					ctx.win.SetFullscreen(0)
				} else {
					ctx.win.SetFullscreen(sdl.WINDOW_FULLSCREEN)
				}
				fullscreen = !fullscreen
			case DRAW_TOGGLE_WIREFRAME:
				ctx.wireframe = !ctx.wireframe
			}
		default:
			/* ignore */
		}

		tickdelta := int64(sdl.GetTicks()) - int64(ticks_start)
		if tickdelta <= 0 {
			tickdelta = 1
		}
		tpf = tickdelta
		tickdelta = ticks_per_frame - tickdelta
		if tickdelta <= 0 {
			tickdelta = 1
		}

		sdl.Delay(uint32(tickdelta))
	}
}
