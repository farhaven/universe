package orrery

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"../vector"
)

type Planet struct {
	T   float64
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	L sync.Mutex
}

type command interface{}
type CommandSpawnPlanet struct {
	Pos vector.V3
}
type CommandSpawnVolume struct {
	Pos vector.V3
}
type CommandPause struct{}
type Orrery struct {
	planets     []*Planet
	trailLength int
	q           chan bool
	l           sync.Mutex
	c           chan command
	looptime    time.Duration
	Paused      bool
}

func (o *Orrery) Planets() []*Planet {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Planet, len(o.planets))
	copy(r, o.planets)

	return r
}

func (p Planet) String() string {
	return fmt.Sprintf(`T: %0.2f R:%.2f, M:%.2f, Pos:%s, Vel:%s`, p.T, p.R, p.M, p.Pos, p.Vel)
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

func (p *Planet) applyForce(f vector.V3, s float64) {
	p.Vel = p.Vel.Add(f.Scaled(s / p.M))
}

func (p *Planet) collide(px *Planet) {
	if p == px {
		panic(`can't collide with myself!`)
	}

	if p.M == 0 || px.M == 0 {
		panic(`colliding a planet with zero mass!`)
	}

	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	d := p.Pos.Distance(px.Pos)
	if d > p.R+px.R {
		return
	}
	if d < math.Max(p.R, px.R) {
		/* XXX: Merge planets */
		return
	}

	CR := 0.1

	a1 := 2 * px.M / (p.M + px.M)
	d1 := p.Pos.Sub(px.Pos)
	dbar1 := d1.X * d1.X + d1.Y * d1.Y + d1.Z * d1.Z
	v1 := p.Vel.Sub(px.Vel).Dot(d1) / dbar1

	a2 := 2 * p.M / (p.M + px.M)
	d2 := px.Pos.Sub(p.Pos)
	dbar2 := d2.X * d2.X + d2.Y * d2.Y + d2.Z * d2.Z
	v2 := px.Vel.Sub(p.Vel).Dot(d2) / dbar2

	p.applyForce(d1, CR*a1*v1)
	px.applyForce(d2, CR*a2*v2)

	p.T += (a1 * (1 - CR)) / p.M
	px.T += (a2 * (1 - CR)) / px.M
}

func (p *Planet) interactGravity(px *Planet) {
	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	if p.M == 0 || px.M == 0 {
		return
	}

	if px == p {
		panic(`can't gravitationally interact with myself!`)
	}

	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)

	v := px.Pos.Sub(p.Pos)

	d := math.Max(1, v.Length())

	M := p.M + px.M
	a := (G * M) / (d * d)

	v = v.Normalized().Scaled(a)

	p.applyForce(v, 1)
	px.applyForce(v, -1)
}

func (o *Orrery) loop() {
	for {
		select {
		case c := <-o.c:
			switch c := c.(type) {
			case CommandSpawnPlanet:
				o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 5, Pos: c.Pos})
			case CommandSpawnVolume:
				rn := func(r float64) float64 {
					return (rand.Float64() - 0.5) * r
				}

				for i := 0; i < 10; i++ {
					px := vector.V3{c.Pos.X + rn(100), c.Pos.Y + rn(100), c.Pos.Z + rn(100)}
					o.planets = append(o.planets, &Planet{T: 0, R: 1.0, M: 2, Pos: px})
				}
			case CommandPause:
				o.Paused = !o.Paused
			default:
				panic(fmt.Sprintf(`unknown orrery command: %T %v`, c, c))
			}
		default:
		}

		if o.Paused {
			time.Sleep(o.looptime)
			continue
		}

		t_start := time.Now()
		o.l.Lock()

		pchan := make(chan [2]*Planet)
		wg := sync.WaitGroup{}
		gw := func() {
			for p := range pchan {
				p[0].interactGravity(p[1])
				wg.Done()
			}
		}
		for i := 0; i < 4; i++ {
			go gw()
		}
		for i, p := range o.planets {
			for _, px := range o.planets[i+1:] {
				wg.Add(1)
				pchan <- [2]*Planet{p, px}
			}
		}
		wg.Wait()

		for _, p := range o.planets {
			p.move(o.trailLength)
		}

		// Check for collisions
		for i, p := range o.planets {
			for _, px := range o.planets[i+1:] {
				p.collide(px)
			}
		}
		o.l.Unlock()

		t_sleep := o.looptime.Nanoseconds() - time.Since(t_start).Nanoseconds()
		if t_sleep > 0 {
			time.Sleep(time.Duration(t_sleep) * time.Nanosecond)
		}
	}
}

func (o *Orrery) QueueCommand(c command) {
	o.c <- c
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
		c: make(chan command, 20),
	}

	go o.loop()

	return o
}
