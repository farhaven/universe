package main

import (
	"log"

	"./orrery"

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

	o := orrery.New()

	width, height := 1024, 768
	camera := NewCamera(width, height, -40, 40, 10)
	sdl.SetRelativeMouseMode(true)
	go camera.handleCommands()

	ctx := NewDrawContext(width, height, fnt, camera, o)

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
			if e.State == sdl.RELEASED {
				continue
			}
			switch e.Button {
			case 1:
				o.SpawnPlanet(camera.pos.X, camera.pos.Y, camera.pos.Z)
			}
		case *sdl.KeyDownEvent:
			switch getNameFromKeysym(e.Keysym) {
			case `Q`:
				ctx.queueCommand(DRAW_QUIT)
				return
			case `F`:
				ctx.queueCommand(DRAW_FULLSCREEN)
			case `1`:
				ctx.queueCommand(DRAW_TOGGLE_WIREFRAME)
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
				camera.queueCommand(CAMERA_TURN, 0, int32(-10))
			case `Down`:
				camera.queueCommand(CAMERA_TURN, 0, int32(+10))
			default:
				log.Printf(`key press: %v %s`, e.Type, getNameFromKeysym(e.Keysym))
			}
		case *sdl.QuitEvent:
			ctx.queueCommand(DRAW_QUIT)
			return
		}
	}
}
