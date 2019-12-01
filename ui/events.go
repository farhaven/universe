package ui

import (
	"log"
	"time"

	"git.c3pb.de/farhaven/universe/orrery"
	"git.c3pb.de/farhaven/universe/vector"

	"github.com/go-gl/glfw/v3.1/glfw"
)

func (ctx *DrawContext) EventLoop(o *orrery.Orrery, shutdown chan struct{}) {
	ctx.win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyQ:
			ctx.QueueCommand(DRAW_QUIT)
		case glfw.KeyF:
			ctx.QueueCommand(DRAW_FULLSCREEN)
		case glfw.Key1:
			ctx.QueueCommand(DRAW_TOGGLE_WIREFRAME)
		case glfw.KeyH:
			ctx.QueueCommand(DRAW_TOGGLE_VERBOSE)
		case glfw.KeyV:
			o.QueueCommand(orrery.CommandSpawnVolume{ctx.cam.Pos})
		case glfw.KeyB:
			o.QueueCommand(orrery.CommandSpawnVolume{vector.V3{}})
		case glfw.KeyN:
			o.QueueCommand(orrery.CommandSpawnParticle{Pos: vector.V3{}})
		case glfw.KeySpace:
			ctx.cam.QueueCommand(cameraCommandReset{})
		case glfw.KeyP:
			o.QueueCommand(orrery.CommandPause{})
		case glfw.KeyJ:
			if mods&glfw.ModShift != 0 {
				o.QueueCommand(orrery.CommandStore{})
			} else if mods == 0 {
				o.QueueCommand(orrery.CommandLoad{})
			}
		default:
			log.Printf(`key: key:%v s:%v a:%v m:%v`, key, scancode, action, mods)
		}
	})

	cursorx, cursory := float64(0), float64(0)
	ctx.win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	ctx.win.SetCursorPosCallback(func(w *glfw.Window, xpos float64, ypos float64) {
		xrel := cursorx - xpos
		yrel := cursory - ypos
		ctx.cam.QueueCommand(cameraCommandTurn{float64(xrel), float64(-yrel)})
		cursorx, cursory = xpos, ypos
	})
	ctx.win.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		ctx.cam.QueueCommand(cameraCommandMove{0, int32(-yoff * 10)})
	})
	ctx.win.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		if button == 0 {
			o.QueueCommand(orrery.CommandSpawnParticle{Pos: ctx.cam.Pos})
		} else {
			log.Printf(`mouse btn: button:%v action:%v mod:%v`, button, action, mod)
		}
	})

	for {
		if ctx.win.ShouldClose() {
			ctx.QueueCommand(DRAW_QUIT)
			return
		}

		select {
		case <-ctx.shutdown:
			close(shutdown)
			return
		default:
		}

		cameraCommands := map[glfw.Key]cameraCommand{
			glfw.KeyW:     cameraCommandMove{0, 1},
			glfw.KeyS:     cameraCommandMove{0, -1},
			glfw.KeyA:     cameraCommandMove{1, 0},
			glfw.KeyD:     cameraCommandMove{-1, 0},
			glfw.KeyLeft:  cameraCommandTurn{10, 0},
			glfw.KeyRight: cameraCommandTurn{-10, 0},
			glfw.KeyUp:    cameraCommandTurn{0, -10},
			glfw.KeyDown:  cameraCommandTurn{0, 10},
		}
		for k, cmd := range cameraCommands {
			if ctx.win.GetKey(k) == glfw.Press {
				ctx.cam.QueueCommand(cmd)
			}
		}

		time.Sleep(5 * time.Millisecond)
	}
}
