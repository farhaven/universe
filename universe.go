package main

import (
	"log"
	"runtime"
	"math"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/go-gl/gl"
	"github.com/go-gl/glu"
	/*
	"github.com/lucasb-eyer/go-colorful"
	*/
)

type DrawCommand int
const (
	DRAW_QUIT = iota
	DRAW_FULLSCREEN
)

const (
	CAMERA_TURN = iota
	CAMERA_MOVE
)
type CameraCommand struct {
	Type int
	X int32
	Y int32
}
type Camera struct {
	cmds chan CameraCommand

	screenw int
	screenh int

	x float64
	y float64
	z float64

	alpha float64
	theta float64
}

func NewCamera(width, height int, x, y float64) *Camera {
	c := &Camera{
		cmds: make(chan CameraCommand),
		screenw: width, screenh: height,
		x: x, y: y,
	}
	sdl.SetRelativeMouseMode(true)
	return c
}
func (c *Camera) Update() {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()

	ratio := float64(c.screenw) / float64(c.screenh)
	glu.Perspective(60, ratio, 0.5, float64(c.screenw))

	vx := math.Cos(c.theta) * 10 + c.x
	vy := math.Sin(c.theta) * 10 + c.y
	vz := math.Sin(c.alpha) * 10 + c.z

	glu.LookAt(c.x, c.y, c.z, vx, vy, vz, 0, 0, 1)
}

func (c *Camera) handleCommands() {
	for cmd := range c.cmds {
		log.Printf(`got camera command %v`, cmd)
		switch cmd.Type {
		case CAMERA_TURN:
			if cmd.X != 0 {
				c.theta += float64(cmd.X) / (float64(c.screenw) / (math.Pi / 2))
				for c.theta < 0 {
					c.theta += 2 * math.Pi
				}
				for c.theta > 2 * math.Pi {
					c.theta -= 2 * math.Pi
				}
			} else if cmd.Y != 0 {
				h := float64(c.screenh) / 2
				c.alpha = (float64(int32(c.screenh) - cmd.Y) - h) / h * math.Pi / 4
			}
			log.Printf(`θ: %v α: %f`, c.theta, c.alpha)
		case CAMERA_MOVE:
			if cmd.Y != 0 {
				c.x += float64(cmd.Y) * math.Cos(c.theta)
				c.y += float64(cmd.Y) * math.Sin(c.theta)
			} else if cmd.X != 0 {
				c.x += float64(cmd.X) * math.Cos((c.theta + math.Pi / 2))
				c.y += float64(cmd.X) * math.Sin((c.theta + math.Pi / 2))
			}

			log.Printf(`x: %f y: %f`, c.x, c.y)
		}
	}
}
func (c *Camera) queueCommand(type_ int, x, y int32) {
	c.cmds <-CameraCommand{ type_, x, y }
}

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Keycode(k.Sym))
}

func initScreen(width, height int) (*sdl.Window, *sdl.Renderer) {
	w, r, err := sdl.CreateWindowAndRenderer(width, height, sdl.WINDOW_OPENGL | sdl.WINDOW_INPUT_GRABBED)
	if err != nil {
		log.Fatalf(`can't create window: %s`, err)
	}

	log.Printf(`sdl win: %v, renderer: %v`, w, r)

	if gl.Init() != 0 {
		log.Fatalln(`can't init GL`)
	}

	gl.ClearColor(0.1, 0.1, 0.1, 0.1)

	return w, r
}

func drawSphere(radius float64, lat, lon int) {
	for i := 0; i <= lat; i++ {
		lat0 := math.Pi * (-0.5 + float64(i - 1) / float64(lat))
		z0 := math.Sin(lat0)
		zr0 := math.Cos(lat0)

		lat1 := math.Pi * (-0.5 + float64(i) / float64(lat))
		z1 := math.Sin(lat1)
		zr1 := math.Cos(lat1)

		gl.Begin(gl.QUAD_STRIP)
		for j := 0; j <= lon; j++ {
			lng := 2 * math.Pi * (float64(j - 1) / float64(lon))
			x := math.Cos(lng)
			y := math.Sin(lng)

			gl.Normal3f(float32(x * zr0), float32(y * zr0), float32(z0))
			gl.Vertex3f(float32(x * zr0), float32(y * zr0), float32(z0))
			gl.Normal3f(float32(x * zr1), float32(y * zr1), float32(z1))
			gl.Vertex3f(float32(x * zr1), float32(y * zr1), float32(z1))
		}
		gl.End()
	}
}

func drawScreen(width, height int, cam *Camera, commands chan DrawCommand) {
	/* SDL wants to run on the 'main thread' */
	runtime.LockOSThread()

	fullscreen := false
	w, r := initScreen(width, height)

	gl.Viewport(0, 0, width, height)
	gl.Hint(gl.PERSPECTIVE_CORRECTION_HINT, gl.NICEST)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, 0)

	for {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		/* Do drawing */
		cam.Update()
		drawSphere(1.0, 10, 10)
		// sdl.GL_SwapBuffers()
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

		sdl.Delay(1)
	}
}

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Fatalf(`SDL Init failed: %s`, err)
	}
	defer sdl.Quit()
	n, err := sdl.GetNumVideoDrivers()
	if err != nil {
		log.Fatalf(`can't get number of video drivers: %s`, err)
	}
	log.Printf(`got %d video drivers`, n)
	for i := 0; i < n; i++ {
		log.Printf(`driver %d: %s`, i, sdl.GetVideoDriver(i))
	}
	if err := sdl.VideoInit(sdl.GetVideoDriver(0)); err != nil {
		log.Fatalf(`can't init video: %s`, err)
	}
	defer sdl.VideoQuit()

	width, height := 800, 600
	yoff := float64(height) / 2

	camera := NewCamera(width, height, 0, -5)
	go camera.handleCommands()

	draw_cmd := make(chan DrawCommand)
	go drawScreen(width, height, camera, draw_cmd)

	events := make(chan sdl.Event)
	go func() {
		for {
			events <- sdl.WaitEvent()
		}
	}()

	log.Printf("here we go")

	for e := range events {
		switch e := e.(type) {
		default:
			log.Printf(`event %T`, e)
		case *sdl.WindowEvent:
			/* ignore */
		case *sdl.MouseMotionEvent:
			camera.queueCommand(CAMERA_TURN, int32(-e.XRel), int32(e.Y))
		case *sdl.MouseButtonEvent:
			log.Printf(`mouse button: %v`, e)
		case *sdl.KeyDownEvent:
			switch getNameFromKeysym(e.Keysym) {
			case `Q`:
				draw_cmd <- DRAW_QUIT
				return
			case `F`:
				draw_cmd <- DRAW_FULLSCREEN
			case `W`:
				camera.queueCommand(CAMERA_MOVE, 0, 1)
			case `S`:
				camera.queueCommand(CAMERA_MOVE, 0, -1)
			case `A`:
				camera.queueCommand(CAMERA_MOVE, 1, 0)
			case `D`:
				camera.queueCommand(CAMERA_MOVE, -1, 0)
			case `Left`:
				camera.queueCommand(CAMERA_TURN, 10, 0)
			case `Right`:
				camera.queueCommand(CAMERA_TURN, -10, 0)
			case `Up`:
				yoff = math.Min(yoff + 10, float64(height))
				camera.queueCommand(CAMERA_TURN, 0, int32(yoff))
			case `Down`:
				yoff = math.Max(yoff - 10, 0)
				camera.queueCommand(CAMERA_TURN, 0, int32(yoff))
			default:
				log.Printf(`key press: %v %s`, e.Type, getNameFromKeysym(e.Keysym))
			}
		case *sdl.QuitEvent:
			draw_cmd <- DRAW_QUIT
			return
		}
	}
}
