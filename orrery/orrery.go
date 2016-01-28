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

	invalid bool
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

		v = v.Normalized().Scaled(a)

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

	pl := []*Planet{}

	// Check for collisions
	for i, p := range o.Planets {
		if p.invalid {
			continue
		}
		for j, px := range o.Planets {
			if i == j || px.invalid {
				continue
			}

			d := p.Pos.Distance(px.Pos)
			if d > p.R + px.R {
				continue
			}

			l := p
			s := px
			if px.M > p.M {
				l = px
				s = p
			}

			l.M += s.M
			l.Vel = l.Vel.Add(s.Vel.Scaled(1/l.M))
			s.invalid = true
		}
	}

	for _, p := range o.Planets {
		if !p.invalid {
			pl = append(pl, p)
		}
	}
	o.Planets = pl
}

func (o *Orrery) SpawnPlanet(x, y, z float64) {
	o.Planets = append(o.Planets, &Planet{R: 1.0, M: 5, Pos: vector.V3{x, y, z}})
}

func New () *Orrery {
	o := &Orrery{ Planets: []*Planet{
		&Planet{R: 30.0, M: 500.972}, // Earth
		&Planet{R: 5, M: 7.3459, Pos: vector.V3{X: 200}, Vel: vector.V3{Y: 0.1}}, // Moon
	},}

	return o
}
