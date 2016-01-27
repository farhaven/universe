package main

import (
	"github.com/go-gl-legacy/gl"
	"./vector"
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

	pos vector.V3

	theta float64
	alpha float64
}

func NewCamera(width, height int, x, y, z float64) *Camera {
	c := &Camera{
		cmds:    make(chan CameraCommand),
		screenw: width, screenh: height,
		pos: vector.V3{x, y, z},
	}
	return c
}

func (c *Camera) lookAt(at vector.V3) {
	up := vector.V3{0, 0, 1}

	fw := at.Sub(c.pos)
	fw.Normalize()

	side := fw.Cross(up)
	side.Normalize()

	up = side.Cross(fw)
	up.Normalize()

	m := [16]float64{
		side.X, up.X, -fw.X, 0,
		side.Y, up.Y, -fw.Y, 0,
		side.Z, up.Z, -fw.Z, 0,
		0, 0, 0, 1,
	}

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadMatrixd(&m)
	gl.Translated(-c.pos.X, -c.pos.Y, -c.pos.Z)
}

func (c *Camera) Update() {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()

	fovY := float64(60)
	aspect := float64(c.screenw) / float64(c.screenh)
	zNear := 0.5
	zFar := float64(1024)

	fH := math.Tan(fovY / 360 * math.Pi) * zNear
	fW := fH * aspect

	gl.Frustum(-fW, fW, -fH, fH, zNear, zFar)

	vx := math.Cos(c.alpha)*10 + c.pos.X
	vy := math.Sin(c.alpha)*10 + c.pos.Y
	vz := c.theta * 10 + c.pos.Z

	c.lookAt(vector.V3{vx, vy, vz})
}

func (c *Camera) handleCommands() {
	Pi2 := math.Pi / 2
	for cmd := range c.cmds {
		switch cmd.Type {
		case CAMERA_TURN:
			if cmd.X != 0 {
				c.alpha += float64(cmd.X) / (float64(c.screenw) / Pi2)
				c.alpha = math.Remainder(c.alpha, 2*math.Pi)
			} else if cmd.Y != 0 {
				c.theta -= float64(cmd.Y) / (float64(c.screenh) / Pi2)
				c.theta = math.Max(-Pi2, math.Min(Pi2, c.theta))
			}
		case CAMERA_MOVE:
			if cmd.Y != 0 {
				c.pos.X += float64(cmd.Y) * math.Cos(c.alpha)
				c.pos.Y += float64(cmd.Y) * math.Sin(c.alpha)
				c.pos.Z += float64(cmd.Y) * c.theta
			} else if cmd.X != 0 {
				c.pos.X += float64(cmd.X) * math.Cos(c.alpha + Pi2)
				c.pos.Y += float64(cmd.X) * math.Sin(c.alpha + Pi2)
			}
		case CAMERA_DROP:
			c.pos.Z = 0
		}
	}
}
func (c *Camera) queueCommand(type_ int, x, y int32) {
	c.cmds <- CameraCommand{type_, x, y}
}
