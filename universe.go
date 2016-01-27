package main

import (
	"log"
	"math"

	"github.com/veandco/go-sdl2/sdl"
	ttf "github.com/veandco/go-sdl2/sdl_ttf"
)

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Keycode(k.Sym))
}

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Fatalf(`SDL Init failed: %s`, err)
	}
	defer sdl.Quit()
	if err := sdl.VideoInit(sdl.GetVideoDriver(0)); err != nil {
		log.Fatalf(`can't init video: %s`, err)
	}
	defer sdl.VideoQuit()

	if err := ttf.Init(); err != nil {
		log.Fatalf(`can't init font system: %s`, err)
	}

	fnt, err := ttf.OpenFont("font.ttf", 12)
	if err != nil {
		log.Fatalf(`can't load font.ttf: %s`, err)
	}

	var mode sdl.DisplayMode
	if err := sdl.GetDesktopDisplayMode(0, &mode); err != nil {
		log.Fatalf(`can't get display mode: %s`, err)
	}

	width, height := int(mode.W), int(mode.H)
	yoff := float64(height) / 2

	setupPlanets()

	camera := NewCamera(width, height, -40, 40, 10)
	sdl.SetRelativeMouseMode(true)
	go camera.handleCommands()

	draw_cmd := make(chan DrawCommand)
	go drawScreen(width, height, fnt, camera, draw_cmd)

	events := make(chan sdl.Event)
	go func() {
		for {
			events <- sdl.WaitEvent()
		}
	}()

	for e := range events {
		switch e := e.(type) {
		default:
			log.Printf(`event %T`, e)
		case *sdl.WindowEvent, *sdl.KeyUpEvent, *sdl.TextInputEvent:
			/* ignore */
		case *sdl.MouseWheelEvent:
			camera.queueCommand(CAMERA_MOVE, 0, -e.Y * 10)
		case *sdl.MouseMotionEvent:
			camera.queueCommand(CAMERA_TURN, int32(-e.XRel), int32(e.YRel))
		case *sdl.MouseButtonEvent:
			log.Printf(`mouse button: %v`, e)
		case *sdl.KeyDownEvent:
			switch getNameFromKeysym(e.Keysym) {
			case `Q`:
				draw_cmd <- DRAW_QUIT
				return
			case `F`:
				draw_cmd <- DRAW_FULLSCREEN
			case `1`:
				draw_cmd <- DRAW_TOGGLE_WIREFRAME
			case `W`:
				camera.queueCommand(CAMERA_MOVE, 0, 1)
			case `S`:
				camera.queueCommand(CAMERA_MOVE, 0, -1)
			case `A`:
				camera.queueCommand(CAMERA_MOVE, 1, 0)
			case `D`:
				camera.queueCommand(CAMERA_MOVE, -1, 0)
			case `Space`:
				camera.queueCommand(CAMERA_DROP, 0, 0)
			case `Left`:
				camera.queueCommand(CAMERA_TURN, 10, 0)
			case `Right`:
				camera.queueCommand(CAMERA_TURN, -10, 0)
			case `Up`:
				yoff = math.Min(yoff+10, float64(height))
				camera.queueCommand(CAMERA_TURN, 0, int32(yoff))
			case `Down`:
				yoff = math.Max(yoff-10, 0)
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
