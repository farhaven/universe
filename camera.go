package main

import (
	"github.com/go-gl-legacy/gl"
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

func NewCamera(width, height int, x, y, z float64) *Camera {
	c := &Camera{
		cmds:    make(chan CameraCommand),
		screenw: width, screenh: height,
		x: x, y: y, z: z,
	}
	return c
}

type Vector3 [3]float64
func (v *Vector3) length() float64 {
	return math.Sqrt(v[0] * v[0] + v[1] * v[1] + v[2] * v[2])
}
func (v *Vector3) cross(o Vector3) Vector3 {
	return Vector3{
		v[1] * o[2] - o[1] * v[2],
		o[0] * v[2] - v[0] * o[2],
		v[0] * o[1] - o[0] * v[1],
	}
}
func (v *Vector3) normalize() {
	l := v.length()
	v[0] /= l
	v[1] /= l
	v[2] /= l
}

func (c *Camera) lookAt(at [3]float64) {
	up := Vector3{0, 0, 1}

	fw := Vector3{ at[0] - c.x, at[1] - c.y, at[2] - c.z }
	fw.normalize()

	side := fw.cross(up)
	side.normalize()

	up = side.cross(fw)
	up.normalize()

	m := [16]float64{
		side[0], up[0], -fw[0], 0,
		side[1], up[1], -fw[1], 0,
		side[2], up[2], -fw[2], 0,
		0, 0, 0, 1,
	}

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadMatrixd(&m)
	gl.Translated(-c.x, -c.y, -c.z)
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

	vx := math.Cos(c.alpha)*10 + c.x
	vy := math.Sin(c.alpha)*10 + c.y
	vz := c.theta * 10 + c.z

	c.lookAt(Vector3{vx, vy, vz})
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
				c.x += float64(cmd.Y) * math.Cos(c.alpha)
				c.y += float64(cmd.Y) * math.Sin(c.alpha)
				c.z += float64(cmd.Y) * c.theta
			} else if cmd.X != 0 {
				c.x += float64(cmd.X) * math.Cos(c.alpha + Pi2)
				c.y += float64(cmd.X) * math.Sin(c.alpha + Pi2)
			}
		case CAMERA_DROP:
			c.z = 0
		}
	}
}
func (c *Camera) queueCommand(type_ int, x, y int32) {
	c.cmds <- CameraCommand{type_, x, y}
}
