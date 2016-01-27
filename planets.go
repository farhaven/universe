package main

import (
	"math"

	"github.com/go-gl/gl"
	"github.com/lucasb-eyer/go-colorful"
)

type Vector3 struct {
	x, y, z float64
}

func (v *Vector3) add (o Vector3) {
	v.x += o.x
	v.y += o.y
	v.z += o.z
}

func (v *Vector3) sub (o Vector3) {
	v.x -= o.x
	v.y -= o.y
	v.z -= o.z
}

func (v *Vector3) scale (n float64) {
	v.x *= n
	v.y *= n
	v.z *= n
}

func (v *Vector3) scaled (n float64) Vector3 {
	return Vector3{ v.x * n, v.y * n, v.z * n }
}

func (v *Vector3) length() float64 {
	return math.Sqrt(v.x * v.x + v.y * v.y + v.z * v.z)
}

type Planet struct {
	r   float64
	m   float64
	pos Vector3
	vel Vector3
}

var planets []*Planet

func (p *Planet) draw(wireframe bool) {
	c := colorful.Hcl(math.Remainder((math.Pi / p.m)*360, 360), 0.9, 0.9)

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()

	gl.Translated(p.pos.x, p.pos.y, p.pos.z)
	gl.Scaled(p.r, p.r, p.r)

	gl.Color3f(float32(c.R), float32(c.G), float32(c.B))

	drawUnitSphere(10, 10, wireframe)

	gl.PopMatrix()
}

func (p *Planet) move() {
	p.pos.add(p.vel)
}

func (p *Planet) affectGravity() {
	// G := 6.67 * math.Pow(10, -11)
	G := float64(.1)
	for _, px := range planets {
		if p == px {
			continue
		}

		v := Vector3{ px.pos.x, px.pos.y, px.pos.z }
		v.sub(p.pos)

		d := math.Max(1, v.length())

		M := p.m + px.m
		a := (G * M) / (d * d)

		v.scale(a/v.length())

		p.vel.add(v.scaled(1/p.m))
	}
}

func drawPlanets(wireframe bool) {
	for _, p := range planets {
		p.draw(wireframe)
	}
}

func stepPlanets() {
	for _, p := range planets {
		p.affectGravity()
	}
	for _, p := range planets {
		p.move()
	}
}

func spawnPlanet(x, y, z float64) {
	planets = append(planets, &Planet{r: 1.0, pos: Vector3{x, y, z}})
}

func setupPlanets() {
	planets = []*Planet{
		&Planet{r: 30.0, m: 500.972}, // Earth
		&Planet{r: 5, m: 7.3459, pos: Vector3{x: 400}, vel: Vector3{y: 0.1}}, // Moon
	}

	go stepPlanets()
}
