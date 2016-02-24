package orrery

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"../vector"
)

type Planet struct {
	T   uint64
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	L sync.Mutex
}

type Orrery struct {
	planets     []*Planet
	trailLength int
	q           chan bool
	l           sync.Mutex
	looptime    time.Duration
}

func (o *Orrery) Planets() []*Planet {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Planet, len(o.planets))
	copy(r, o.planets)

	return r
}

func (p *Planet) move(trailLength int) {
	p.L.Lock()
	defer p.L.Unlock()

	newPos := p.Pos.Add(p.Vel)

	addToTrail := false

	if len(p.Trail) > 0 {
		d := newPos.Distance(p.Trail[len(p.Trail)-1])

		if d > p.R {
			addToTrail = true
		}
	} else {
		addToTrail = true
	}

	if addToTrail {
		p.Trail = append(p.Trail, p.Pos)
		if len(p.Trail) > trailLength {
			p.Trail = p.Trail[len(p.Trail)-trailLength:]
		}
	}

	p.Pos = newPos
}

func (p *Planet) applyForce(f vector.V3) {
	p.Vel = p.Vel.Add(f.Scaled(1/p.M))
}

func (p *Planet) collide(px *Planet) {
	/* Totally elastic collision, no transformation of kinetic energy to heat or rotational energy, no mass transfer */
	/* c.f. https://en.m.wikipedia.org/wiki/Elastic_collision */
	/* Derived from the formula for a collision between two moving objects on a 2D plane */

	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	d := p.Pos.Distance(px.Pos)
	if d > p.R+px.R {
		return
	}

	/* V1 */
	a1 := -2 * px.M / (p.M + px.M)
	d1 := p.Pos.Sub(px.Pos)
	p.applyForce(d1.Scaled(a1 * (p.Vel.Sub(px.Vel).Dot(d1) / d1.Length())))

	/* V2 */
	a2 := -2 * p.M / (p.M + px.M)
	d2 := px.Pos.Sub(p.Pos)
	px.applyForce(d2.Scaled(a2 * (px.Vel.Sub(p.Vel).Dot(d2) / d2.Length())))
}

func (p *Planet) affectGravity(o *Orrery) {
	p.L.Lock()
	defer p.L.Unlock()

	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)
	for _, px := range o.planets {
		if p == px {
			continue
		}

		v := px.Pos.Sub(p.Pos)

		d := math.Max(1, v.Length())

		M := px.M
		a := (G * M) / (d * d)

		v = v.Normalized().Scaled(a)

		p.applyForce(v)
	}
}

func (o *Orrery) loop() {
	for {
		t_start := time.Now()
		o.l.Lock()

		wg := sync.WaitGroup{}
		wg.Add(len(o.planets))
		for _, p := range o.planets {
			p := p
			go func() {
				defer wg.Done()
				p.affectGravity(o)
			}()
		}
		wg.Wait()

		for _, p := range o.planets {
			p.move(o.trailLength)
		}

		// Check for collisions
		for i, p := range o.planets[:len(o.planets)-1] {
			for _, px := range o.planets[i+1:] {
				if c := p.collide(px); c == TOTAL {
					l, s := p, px
					if l.M < s.M {
						l, s = px, p
					}
					s.Pos = l.Pos
					s.Vel = s.Vel
				}
			}
		}

		o.l.Unlock()

		t_sleep := o.looptime.Nanoseconds() - time.Since(t_start).Nanoseconds()
		if t_sleep > 0 {
			time.Sleep(time.Duration(t_sleep) * time.Nanosecond)
		}
	}
}

func (o *Orrery) SpawnPlanet(p vector.V3) {
	o.l.Lock()
	defer o.l.Unlock()

	o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 5, Pos: p})
}

func (o *Orrery) SpawnVolume(p vector.V3) {
	o.l.Lock()
	defer o.l.Unlock()

	rn := func(r float64) float64 {
		return (rand.Float64() - 0.5) * r
	}

	for i := 0; i < 10; i++ {
		px := vector.V3{p.X + rn(100), p.Y + rn(100), p.Z + rn(100)}
		o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 2, Pos: px})
	}
}

func New() *Orrery {
	o := &Orrery{
		planets: []*Planet{
			&Planet{T: 0, R: 30.0, M: 500.972},                                             // Earth
			&Planet{T: 0, R: 5, M: 7.3459, Pos: vector.V3{X: 200}, Vel: vector.V3{Y: 0.1}}, // Moon
		},
		trailLength: 20,
		looptime:    5 * time.Millisecond,

		q: make(chan bool),
	}

	go o.loop()

	return o
}
