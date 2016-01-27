package main

import (
	"github.com/go-gl/gl"
	"github.com/go-gl/glu"
	"math"
)

const (
	CAMERA_TURN = iota
	CAMERA_MOVE
	CAMERA_DROP
)

type CameraCommand struct {
	Type int
	X    int32
	Y    int32
}
type Camera struct {
	cmds chan CameraCommand

	screenw int
	screenh int

	x float64
	y float64
	z float64

	theta float64
	alpha float64
}

func NewCamera(width, height int, x, y float64) *Camera {
	c := &Camera{
		cmds:    make(chan CameraCommand),
		screenw: width, screenh: height,
		x: x, y: y,
	}
	return c
}
func (c *Camera) Update() {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()

	ratio := float64(c.screenw) / float64(c.screenh)
	glu.Perspective(60, ratio, 0.5, float64(c.screenw))

	vx := math.Cos(c.alpha)*10 + c.x
	vy := math.Sin(c.alpha)*10 + c.y
	vz := c.theta * 10 + c.z

	glu.LookAt(c.x, c.y, c.z, vx, vy, vz, 0, 0, 1)
}

func (c *Camera) handleCommands() {
	for cmd := range c.cmds {
		switch cmd.Type {
		case CAMERA_TURN:
			if cmd.X != 0 {
				c.alpha += float64(cmd.X) / (float64(c.screenw) / (math.Pi / 2))
				for c.alpha < 0 {
					c.alpha += 2 * math.Pi
				}
				for c.alpha > 2*math.Pi {
					c.alpha -= 2 * math.Pi
				}
			} else if cmd.Y != 0 {
				h := float64(c.screenh) / 2
				c.theta = ((float64(int32(c.screenh)-cmd.Y) - h) / h) * math.Pi / 2
			}
		case CAMERA_MOVE:
			if cmd.Y != 0 {
				c.x += float64(cmd.Y) * math.Cos(c.alpha)
				c.y += float64(cmd.Y) * math.Sin(c.alpha)
				c.z += float64(cmd.Y) * c.theta
			} else if cmd.X != 0 {
				c.x += float64(cmd.X) * math.Cos((c.alpha + math.Pi/2))
				c.y += float64(cmd.X) * math.Sin((c.alpha + math.Pi/2))
			}
		case CAMERA_DROP:
			c.z = 0
		}
	}
}
func (c *Camera) queueCommand(type_ int, x, y int32) {
	c.cmds <- CameraCommand{type_, x, y}
}
