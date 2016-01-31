package ui

import (
	"../orrery"

	"log"
	"github.com/veandco/go-sdl2/sdl"
)

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Keycode(k.Sym))
}

func (ctx *DrawContext) EventLoop(o *orrery.Orrery) {
	for {
		switch e := sdl.WaitEvent().(type) {
		default:
			log.Printf(`event %T`, e)
		case *sdl.WindowEvent, *sdl.KeyUpEvent, *sdl.TextInputEvent:
			/* ignore */
		case *sdl.MouseWheelEvent:
			ctx.cam.QueueCommand(CAMERA_MOVE, 0, -e.Y*10)
		case *sdl.MouseMotionEvent:
			ctx.cam.QueueCommand(CAMERA_TURN, int32(-e.XRel), int32(e.YRel))
		case *sdl.MouseButtonEvent:
			if e.State == sdl.RELEASED {
				continue
			}
			switch e.Button {
			case 1:
				o.SpawnPlanet(ctx.cam.Pos.X, ctx.cam.Pos.Y, ctx.cam.Pos.Z)
			}
		case *sdl.KeyDownEvent:
			switch getNameFromKeysym(e.Keysym) {
			case `Q`:
				ctx.QueueCommand(DRAW_QUIT)
				return
			case `F`:
				ctx.QueueCommand(DRAW_FULLSCREEN)
			case `1`:
				ctx.QueueCommand(DRAW_TOGGLE_WIREFRAME)
			case `W`:
				ctx.cam.QueueCommand(CAMERA_MOVE, 0, 1)
			case `S`:
				ctx.cam.QueueCommand(CAMERA_MOVE, 0, -1)
			case `A`:
				ctx.cam.QueueCommand(CAMERA_MOVE, 1, 0)
			case `D`:
				ctx.cam.QueueCommand(CAMERA_MOVE, -1, 0)
			case `Space`:
				ctx.cam.QueueCommand(CAMERA_DROP, 0, 0)
			case `Left`:
				ctx.cam.QueueCommand(CAMERA_TURN, 10, 0)
			case `Right`:
				ctx.cam.QueueCommand(CAMERA_TURN, -10, 0)
			case `Up`:
				ctx.cam.QueueCommand(CAMERA_TURN, 0, int32(-10))
			case `Down`:
				ctx.cam.QueueCommand(CAMERA_TURN, 0, int32(+10))
			case `P`:
				panic("User requested panic")
			default:
				log.Printf(`key press: %v %s`, e.Type, getNameFromKeysym(e.Keysym))
			}
		case *sdl.QuitEvent:
			ctx.QueueCommand(DRAW_QUIT)
			return
		}
	}
}
