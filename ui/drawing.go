package ui

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"reflect"
	"runtime"
	"time"
	"unsafe"

	"../orrery"
	"../vector"
	"./text"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/lucasb-eyer/go-colorful"
)

type DrawCommand int

const (
	DRAW_QUIT = iota
	DRAW_FULLSCREEN
	DRAW_TOGGLE_WIREFRAME
)

type DrawContext struct {
	width, height int
	win           *glfw.Window
	cmd           chan DrawCommand

	cam *Camera

	wireframe bool

	txt      *text.Context
	shutdown chan struct{}

	spheresWireframe map[int]uint32
	spheresSolid map[int]uint32

	listId uint32 // ID of the next free call list
}

func NewDrawContext(width, height int, o *orrery.Orrery) DrawContext {
	txt, err := text.NewContext("font.ttf")
	if err != nil {
		log.Fatalf(`can't create text context: %s`)
	}

	cam := NewCamera(width, height, -40, 40, 10)

	c := make(chan DrawContext)

	// This is a hack to make sure all drawing stuff runs in the same goroutine
	// XXX: This should probably be replaced by a goroutine that listens to a channel for GL commands
	go func() {
		/* SDL and GL want to run on the 'main thread' */
		runtime.LockOSThread()

		log.Println(`initializing glfw`)
		if err := glfw.Init(); err != nil {
			log.Fatalln(err)
		}

		glfw.WindowHint(glfw.Resizable, 0)
		glfw.WindowHint(glfw.Floating, 1)
		// w, err := glfw.CreateWindow(width, height, "Universe", glfw.GetPrimaryMonitor(), nil)
		w, err := glfw.CreateWindow(int(width), int(height), "Universe", nil, nil)
		if err != nil {
			log.Fatalf(`can't create window: %s`, err)
		}
		defer w.Destroy()

		w.MakeContextCurrent()

		if err := gl.Init(); err != nil {
			log.Fatalln(`can't init GL: %s`, err)
		}

		gl.ClearColor(0.1, 0.1, 0.1, 0.1)
		gl.Enable(gl.DEPTH_TEST)

		ctx := DrawContext{
			width: width, height: height,
			win:       w,
			cmd:       make(chan DrawCommand),
			wireframe: true,
			cam:       cam,
			txt:       txt,
			shutdown:  make(chan struct{}),
			spheresWireframe: make(map[int]uint32),
			spheresSolid: make(map[int]uint32),
		}
		c <- ctx

		eventShutdown := make(chan struct{})
		go ctx.EventLoop(o, eventShutdown)
		ctx.drawScreen(o)
		close(ctx.shutdown)
		<-eventShutdown
	}()

	return <-c
}

func (ctx *DrawContext) WaitForShutdown() {
	<-ctx.shutdown
}

func (ctx *DrawContext) Shutdown() {
	glfw.Terminate()
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
	p.L.Lock()
	defer p.L.Unlock()

	c := colorful.Hcl(math.Remainder((math.Pi/p.M)*360, 360), 0.9, 0.9)

	ctx.drawSphere(p.Pos, p.R, c)
	for i, pos := range p.Trail {
		ctx.drawSphere(pos, 1/float64(len(p.Trail)-i+1), c)
	}
}

func (ctx *DrawContext) drawSphere(p vector.V3, r float64, c colorful.Color) {
	/* TODO:
	   - decrease sphere detail if it's further away
	   - only draw spheres that would be visible inside the frustum:
	     - (no small spheres near the far plane)
	*/
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

	l, ok := uint32(0), false
	if ctx.wireframe {
		l, ok = ctx.spheresWireframe[slices]
	} else {
		l, ok = ctx.spheresSolid[slices]
	}

	if !ok {
		ctx.listId++ // XXX: atomic?
		l = ctx.listId
		gl.NewList(l, gl.COMPILE)
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
		gl.EndList()
		if ctx.wireframe {
			ctx.spheresWireframe[slices] = l
		} else {
			ctx.spheresSolid[slices] = l
		}
	}

	gl.CallList(l)
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

func (ctx *DrawContext) createHudTexture(o *orrery.Orrery, frametime time.Duration) (uint32, [2]int, error) {
	lines := []string{
		"WASD: Move, 1: Toggle wireframe, F: Fullscreen, Q: Quit",
		"Mouse Wheel: Move fast, Mouse Btn #1: Spawn planet",
		"P: panic and dump stacks",
		fmt.Sprintf(` α: %0.2f θ: %0.2f`, ctx.cam.alpha, ctx.cam.theta),
		fmt.Sprintf(` x: %0.2f y: %0.2f z: %0.2f`, ctx.cam.Pos.X, ctx.cam.Pos.Y, ctx.cam.Pos.Z),
		fmt.Sprintf(` Last frame time: %s`, frametime),
	}

	for i, p := range o.Planets() {
		p.L.Lock()
		l := fmt.Sprintf(` π %d: r=%0.2f M=%0.2f pos=(%0.2f, %0.2f, %0.2f), vel=(%0.2f, %0.2f, %0.2f)`, i, p.R, p.M, p.Pos.X, p.Pos.Y, p.Pos.Z, p.Vel.X, p.Vel.Y, p.Vel.Z)
		p.L.Unlock()
		lines = append(lines, l)
	}

	var txt uint32
	gl.GenTextures(1, &txt)

	gl.BindTexture(gl.TEXTURE_2D, txt)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_BASE_LEVEL, 0)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAX_LEVEL, 0)

	bg := color.RGBA{0, 0, 0, 0}
	fg := color.RGBA{0, 255, 255, 255}
	img, err := ctx.txt.RenderMultiline(lines, 12.5, bg, fg)
	if err != nil {
		return 0, [2]int{0, 0}, err
	}
	v := reflect.ValueOf(img.Pix)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA,
		int32(img.Bounds().Dx()), int32(img.Bounds().Dy()),
		0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(v.Index(0).UnsafeAddr()))

	return txt, [2]int{img.Bounds().Dx(), img.Bounds().Dy()}, nil
}

func (ctx *DrawContext) drawHud(o *orrery.Orrery, frametime time.Duration) {
	txt, size, err := ctx.createHudTexture(o, frametime)
	if err != nil {
		log.Fatalf(`can't create texture from text surface: %s`, err)
	}
	defer gl.DeleteTextures(1, &txt)

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0.0, float64(ctx.width), float64(ctx.height), 0.0, -1.0, 1.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Clear(gl.DEPTH_BUFFER_BIT)

	gl.BindTexture(gl.TEXTURE_2D, txt)
	gl.Enable(gl.TEXTURE_2D)
	defer gl.Disable(gl.TEXTURE_2D)

	gl.Color3f(1, 1, 1)
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0, 0)
	gl.Vertex2f(0.0, 0.0)
	gl.TexCoord2f(1, 0)
	gl.Vertex2f(float32(size[0]), 0.0)
	gl.TexCoord2f(1, 1)
	gl.Vertex2f(float32(size[0]), float32(size[1]))
	gl.TexCoord2f(0, 1)
	gl.Vertex2f(0.0, float32(size[1]))
	gl.End()

	gl.PopMatrix()
}

func (ctx *DrawContext) drawScreen(o *orrery.Orrery) {
	fullscreen := false

	gl.Viewport(0, 0, int32(ctx.width), int32(ctx.height))
	gl.Hint(gl.PERSPECTIVE_CORRECTION_HINT, gl.NICEST)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, 0)

	t_delta := time.Duration(0)
	frametime := time.Second / 24
	log.Printf(`ft: %v`, frametime)

	frametimes := time.Duration(0)
	nsamples := 0
	slowest_frame := time.Duration(0)
	frames_over_deadline := 0

	defer func() {
		log.Printf(`average frame time: %v, #sample: %d`, frametimes / time.Duration(nsamples), nsamples)
		log.Printf(`slowest frame: %v, # of frames over %v: %d`, slowest_frame, frametime, frames_over_deadline)
	}()

	for {
		t_start := time.Now()
		o.Step()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		ctx.cam.Update()
		ctx.drawGrid()
		ctx.drawPlanets(o)
		ctx.drawHud(o, t_delta)
		ctx.win.SwapBuffers()

		select {
		case cmd := <-ctx.cmd:
			switch cmd {
			case DRAW_QUIT:
				return
			case DRAW_FULLSCREEN:
				/*
					if fullscreen {
						ctx.win.SetFullscreen(0)
					} else {
						ctx.win.SetFullscreen(sdl.WINDOW_FULLSCREEN)
					}
				*/
				fullscreen = !fullscreen
			case DRAW_TOGGLE_WIREFRAME:
				ctx.wireframe = !ctx.wireframe
			}
		default:
			/* ignore */
		}

		t_delta = time.Since(t_start)
		frametimes += t_delta
		nsamples++
		if t_delta > slowest_frame {
			slowest_frame = t_delta
		}
		if t_delta > frametime {
			frames_over_deadline++
		}
		t_sleep := frametime.Nanoseconds() - t_delta.Nanoseconds()
		if t_sleep > 0 {
			time.Sleep(time.Duration(t_sleep) * time.Nanosecond)
		}
	}
}
