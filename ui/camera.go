package ui

import (
	"../vector"
	"github.com/go-gl-legacy/gl"
	"github.com/veandco/go-sdl2/sdl"
	"log"
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

	Pos vector.V3

	theta float64
	alpha float64

	frustum struct {
		zNear, zFar  float64
		nearH, nearW float64
		farH, farW   float64
		fovY, aspect float64
		planes       []vector.Plane
	}
}

func NewCamera(width, height int, x, y, z float64) *Camera {
	c := &Camera{
		cmds:    make(chan CameraCommand),
		screenw: width, screenh: height,
		Pos: vector.V3{x, y, z},
	}
	c.frustum.zNear = 0.5
	c.frustum.zFar = 1024
	c.frustum.fovY = 60
	c.frustum.aspect = float64(width) / float64(height)

	t := math.Tan(c.frustum.fovY / 360 * math.Pi)
	c.frustum.nearH = t * c.frustum.zNear
	c.frustum.nearW = c.frustum.nearH * c.frustum.aspect
	c.frustum.farH = t * c.frustum.zFar
	c.frustum.farW = c.frustum.farH * c.frustum.aspect

	sdl.SetRelativeMouseMode(true)
	go c.handleCommands()

	return c
}

type FrustumCheckResult int

const (
	INSIDE = iota
	OUTSIDE
	INTERSECT
)

func (r FrustumCheckResult) String() string {
	switch r {
	case INSIDE:
		return "INSIDE"
	case OUTSIDE:
		return "OUTSIDE"
	case INTERSECT:
		return "INTERSECT"
	default:
		log.Fatalf(`Can't get string for unknown frustum check result: %d`, r)
	}

	return ""
}

func (c *Camera) SphereInFrustum(p vector.V3, r float64) FrustumCheckResult {
	rv := FrustumCheckResult(INSIDE)

	for _, pl := range c.frustum.planes {
		d := pl.Distance(p)
		if d < -r {
			return OUTSIDE
		} else if d < r {
			rv = INTERSECT
		}
	}

	return rv
}

func (c *Camera) lookAt(at vector.V3) {
	up := vector.V3{0, 0, 1}

	fw := at.Sub(c.Pos).Normalized()
	side := fw.Cross(up).Normalized()
	up = side.Cross(fw).Normalized()

	m := [16]float64{
		side.X, up.X, -fw.X, 0,
		side.Y, up.Y, -fw.Y, 0,
		side.Z, up.Z, -fw.Z, 0,
		0, 0, 0, 1,
	}

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadMatrixd(&m)
	gl.Translated(-c.Pos.X, -c.Pos.Y, -c.Pos.Z)

	// Update frustum
	nc := c.Pos.Sub(fw.Scaled(-c.frustum.zNear))
	fc := c.Pos.Sub(fw.Scaled(-c.frustum.zFar))

	planes := []vector.Plane{
		vector.Plane{fw, nc},            // NEARP
		vector.Plane{fw.Scaled(-1), fc}, // FARP
	}

	nh, nw := c.frustum.nearH, c.frustum.nearW

	// TOP
	aux := nc.Add(up.Scaled(nh)).Sub(c.Pos).Normalized()
	normal := aux.Cross(side)
	planes = append(planes, vector.Plane{normal, nc.Add(up.Scaled(nh))})

	// BOTTOM
	aux = nc.Sub(up.Scaled(nh)).Sub(c.Pos).Normalized()
	normal = side.Cross(aux)
	planes = append(planes, vector.Plane{normal, nc.Sub(up.Scaled(nh))})

	// LEFT
	aux = nc.Sub(side.Scaled(nw)).Sub(c.Pos).Normalized()
	normal = aux.Cross(up)
	planes = append(planes, vector.Plane{normal, nc.Sub(side.Scaled(nw))})

	// LEFT
	aux = nc.Add(side.Scaled(nw)).Sub(c.Pos).Normalized()
	normal = up.Cross(aux)
	planes = append(planes, vector.Plane{normal, nc.Add(side.Scaled(nw))})

	c.frustum.planes = planes
}

func (c *Camera) Update() {
	// This has to be called in the GL thread
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Frustum(-c.frustum.nearW, c.frustum.nearW, -c.frustum.nearH, c.frustum.nearH, c.frustum.zNear, c.frustum.zFar)

	vx := math.Cos(c.alpha)*10 + c.Pos.X
	vy := math.Sin(c.alpha)*10 + c.Pos.Y
	vz := c.theta*10 + c.Pos.Z

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
				c.Pos.X += float64(cmd.Y) * math.Cos(c.alpha)
				c.Pos.Y += float64(cmd.Y) * math.Sin(c.alpha)
				c.Pos.Z += float64(cmd.Y) * c.theta
			} else if cmd.X != 0 {
				c.Pos.X += float64(cmd.X) * math.Cos(c.alpha+Pi2)
				c.Pos.Y += float64(cmd.X) * math.Sin(c.alpha+Pi2)
			}
		case CAMERA_DROP:
			c.Pos.Z = 0
		}
	}
}
func (c *Camera) QueueCommand(type_ int, x, y int32) {
	c.cmds <- CameraCommand{type_, x, y}
}
