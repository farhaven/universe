package orrery

import (
	"math"
	"../vector"
)

type Planet struct {
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3
}

type Orrery struct {
	Planets []*Planet
}

func (p *Planet) move() {
	p.Pos = p.Pos.Add(p.Vel)
}

func (p *Planet) affectGravity(o *Orrery) {
	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)
	for _, px := range o.Planets {
		if p == px {
			continue
		}

		v := px.Pos.Sub(p.Pos)

		d := math.Max(1, v.Length())

		M := p.M + px.M
		a := (G * M) / (d * d)

		v.Normalize()
		v.Scale(a)

		p.Vel = p.Vel.Add(v.Scaled(1/p.M))
	}
}

func (o *Orrery) Step() {
	for _, p := range o.Planets {
		p.affectGravity(o)
	}

	for _, p := range o.Planets {
		p.move()
	}
}

func (o *Orrery) SpawnPlanet(x, y, z float64) {
	o.Planets = append(o.Planets, &Planet{R: 1.0, Pos: vector.V3{x, y, z}})
}

func New () *Orrery {
	o := &Orrery{ Planets: []*Planet{
		&Planet{R: 30.0, M: 500.972}, // Earth
		&Planet{R: 5, M: 7.3459, Pos: vector.V3{X: 400}, Vel: vector.V3{Y: 0.1}}, // Moon
	},}

	return o
}
