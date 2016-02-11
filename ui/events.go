package ui

import (
	"../orrery"

	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"time"
)

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Keycode(k.Sym))
}

func (ctx *DrawContext) EventLoop(o *orrery.Orrery) {
	sdl.SetEventFilterFunc(func(e sdl.Event) bool {
		switch e.(type) {
		case *sdl.QuitEvent:
			return true
		case *sdl.KeyDownEvent:
			return true
		case *sdl.MouseWheelEvent, *sdl.MouseMotionEvent, *sdl.MouseButtonEvent:
			return true
		}
		return false
	})

	events := make(chan sdl.Event)
	go func() {
		for {
			events <- sdl.WaitEvent()
		}
	}()

	for {
		select {
		case e := <-events:
			switch e := e.(type) {
			default:
				panic(fmt.Sprintf(`unknown event of type %T received`, e))
			case *sdl.MouseWheelEvent:
				ctx.cam.QueueCommand(CAMERA_MOVE, 0, -e.Y*10)
			case *sdl.MouseMotionEvent:
				ctx.cam.QueueCommand(CAMERA_TURN, int32(-e.XRel), int32(e.YRel))
			case *sdl.MouseButtonEvent:
				if e.State == sdl.RELEASED {
					panic(fmt.Sprintf(`unexpected mouse button state %d`, e.State))
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
				case `Space`:
					ctx.cam.QueueCommand(CAMERA_DROP, 0, 0)
				case `P`:
					panic("user requested panic")
				}
			case *sdl.QuitEvent:
				ctx.QueueCommand(DRAW_QUIT)
				return
			}
		case <-time.After(5 * time.Millisecond):
		}
		keys := func() []bool {
			k := sdl.GetKeyboardState()
			r := make([]bool, len(k))
			for i, v := range k {
				if v == 1 {
					r[i] = true
				} else {
					r[i] = false
				}
			}
			return r
		}()
		if keys[sdl.SCANCODE_W] {
			ctx.cam.QueueCommand(CAMERA_MOVE, 0, 1)
		}
		if keys[sdl.SCANCODE_S] {
			ctx.cam.QueueCommand(CAMERA_MOVE, 0, -1)
		}
		if keys[sdl.SCANCODE_A] {
			ctx.cam.QueueCommand(CAMERA_MOVE, 1, 0)
		}
		if keys[sdl.SCANCODE_D] {
			ctx.cam.QueueCommand(CAMERA_MOVE, -1, 0)
		}
		if keys[sdl.SCANCODE_LEFT] {
			ctx.cam.QueueCommand(CAMERA_TURN, 10, 0)
		}
		if keys[sdl.SCANCODE_RIGHT] {
			ctx.cam.QueueCommand(CAMERA_TURN, -10, 0)
		}
		if keys[sdl.SCANCODE_UP] {
			ctx.cam.QueueCommand(CAMERA_TURN, 0, int32(-10))
		}
		if keys[sdl.SCANCODE_DOWN] {
			ctx.cam.QueueCommand(CAMERA_TURN, 0, int32(+10))
		}
	}
}
