package orrery

import (
	"../vector"
	"math"
	"sync"
)

type Planet struct {
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	invalid bool
}

type Orrery struct {
	planets     []*Planet
	trailLength int
	q           chan bool
	l           sync.Mutex
}

func (o *Orrery) Planets() []*Planet {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Planet, len(o.planets))
	copy(r, o.planets)

	return r
}

func (p *Planet) move(trailLength int) {
	newPos := p.Pos.Add(p.Vel)

	addToTrail := false

	if len(p.Trail) > 0 {
		d := newPos.Distance(p.Trail[len(p.Trail) - 1])

		if d > p.R {
			addToTrail = true
		}
	} else {
		addToTrail = true
	}

	if addToTrail {
		p.Trail = append(p.Trail, p.Pos)
		if len(p.Trail) > trailLength {
			p.Trail = p.Trail[len(p.Trail) - trailLength:]
		}
	}

	p.Pos = newPos
}

func (p *Planet) affectGravity(o *Orrery) {
	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)
	for _, px := range o.planets {
		if p == px {
			continue
		}

		v := px.Pos.Sub(p.Pos)

		d := math.Max(1, v.Length())

		M := p.M + px.M
		a := (G * M) / (d * d)

		v = v.Normalized().Scaled(a)

		p.Vel = p.Vel.Add(v.Scaled(1 / p.M))
	}
}

func (o *Orrery) Step() {
	o.q <- true
}

func (o *Orrery) loop() {
	for {
		<-o.q
		o.l.Lock()

		for _, p := range o.planets {
			p.affectGravity(o)
		}

		for _, p := range o.planets {
			p.move(o.trailLength)
		}

		pl := []*Planet{}

		// Check for collisions
		for i, p := range o.planets {
			if p.invalid {
				continue
			}
			for j, px := range o.planets {
				if i == j || px.invalid {
					continue
				}

				d := p.Pos.Distance(px.Pos)
				if d > p.R+px.R {
					continue
				}

				l := p
				s := px
				if px.M > p.M {
					l = px
					s = p
				}

				l.M += s.M
				l.Vel = l.Vel.Add(s.Vel.Scaled(1 / l.M))
				s.invalid = true
			}
		}

		for _, p := range o.planets {
			if !p.invalid {
				pl = append(pl, p)
			}
		}
		o.planets = pl
		o.l.Unlock()
	}
}

func (o *Orrery) SpawnPlanet(x, y, z float64) {
	o.l.Lock()
	defer o.l.Unlock()

	o.planets = append(o.planets, &Planet{R: 1.0, M: 5, Pos: vector.V3{x, y, z}})
}

func New() *Orrery {
	o := &Orrery{
		planets: []*Planet{
			&Planet{R: 30.0, M: 500.972},                                             // Earth
			&Planet{R: 5, M: 7.3459, Pos: vector.V3{X: 200}, Vel: vector.V3{Y: 0.1}}, // Moon
		},
		trailLength: 20,

		q: make(chan bool),
	}

	go o.loop()

	return o
}
