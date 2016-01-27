package main

import (
	"github.com/go-gl/gl"
	"github.com/lucasb-eyer/go-colorful"
)

type Planet struct {
	r float64
	x, y, z float64
}

var planets []Planet

func setupPlanets() {
	planets = []Planet{
		Planet{1.0, 0, 0, 0},
		Planet{2.0, 5, 6, 0},
	}
}

func (p *Planet) draw() {
	c := colorful.Hcl(p.r + 180, 0.9, 0.9)

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()

	gl.Translated(p.x, p.y, p.z)
	gl.Scaled(p.r, p.r, p.r)

	gl.Color3f(float32(c.R), float32(c.G), float32(c.B))

	drawUnitSphere(10, 10)

	gl.PopMatrix()
}

func drawPlanets() {
	for _, p := range planets {
		p.draw()
	}
}
