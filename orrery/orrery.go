package orrery

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"../vector"
)

type Particle struct {
	T   float64
	R   float64
	M   float64
	Pos vector.V3
	Vel vector.V3

	Trail []vector.V3

	L sync.Mutex
}

type command interface{}
type CommandSpawnParticle struct {
	Pos vector.V3
}
type CommandSpawnVolume struct {
	Pos vector.V3
}
type CommandPause struct{}
type Orrery struct {
	particles   []*Particle
	trailLength int
	q           chan bool
	l           sync.Mutex
	c           chan command
	looptime    time.Duration
	Paused      bool
}

func (o *Orrery) Particles() []*Particle {
	o.l.Lock()
	defer o.l.Unlock()

	r := make([]*Particle, len(o.particles))
	copy(r, o.particles)

	return r
}

func (p Particle) String() string {
	return fmt.Sprintf(`T: %0.2f R:%.2f, M:%.2f, Pos:%s, Vel:%s`, p.T, p.R, p.M, p.Pos, p.Vel)
}

func (p *Particle) move(trailLength int) {
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

func (p *Particle) applyForce(f vector.V3, s float64) {
	p.Vel = p.Vel.Add(f.Scaled(s / p.M))
}

func (p *Particle) collide(px *Particle) {
	if p == px {
		panic(`can't collide with myself!`)
	}

	if p.M == 0 || px.M == 0 {
		panic(`colliding a particle with zero mass!`)
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
		/* XXX: Merge particles */
		return
	}

	CR := 0.7

	a1 := 2 * px.M / (p.M + px.M)
	v1 := px.Vel.Sub(p.Vel)

	a2 := 2 * p.M / (p.M + px.M)
	v2 := p.Vel.Sub(px.Vel)

	p.applyForce(v1.Normalized(), a1*CR)
	px.applyForce(v2.Normalized(), a2*CR)

	p.T += (a1 * (1 - CR)) / p.M
	px.T += (a2 * (1 - CR)) / px.M
}

func (p *Particle) interactGravity(px *Particle) {
	if px == p {
		panic(`can't gravitationally interact with myself!`)
	}

	p.L.Lock()
	defer p.L.Unlock()

	px.L.Lock()
	defer px.L.Unlock()

	if p.M == 0 || px.M == 0 {
		return
	}

	// G := 6.67 * math.Pow(10, -11)
	G := float64(0.05)

	v := px.Pos.Sub(p.Pos)

	d := math.Max(1, v.Length())

	M := p.M + px.M
	a := (G * M) / (d * d)

	v = v.Normalized()

	p.applyForce(v, a)
	px.applyForce(v, -a)
}

func (o *Orrery) loop() {
	/* XXX: Use barnes-hut simulation for less processing time: O(n^2) -> O(n log n)
	   - https://en.wikipedia.org/wiki/Barnes%E2%80%93Hut_simulation
	*/

	mScale := 15.0
	rScale := math.Pow(mScale, 1/3)

	for {
		t_start := time.Now()

		select {
		case c := <-o.c:
			switch c := c.(type) {
			case CommandSpawnParticle:
				m := rand.Float64() * mScale
				o.particles = append(o.particles, &Particle{T: 0, R: m * rScale, M: m, Pos: c.Pos})
			case CommandSpawnVolume:
				rn := func(r float64) float64 {
					return (rand.Float64() - 0.5) * r
				}

				for i := 0; i < 10; i++ {
					px := vector.V3{c.Pos.X + rn(300), c.Pos.Y + rn(300), c.Pos.Z + rn(300)}
					m := rand.Float64() * mScale
					o.particles = append(o.particles, &Particle{T: 0, R: m * rScale, M: m, Pos: px})
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

		pchan := make(chan [2]*Particle)
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

		o.l.Lock()
		for i, p := range o.particles {
			for _, px := range o.particles[i+1:] {
				wg.Add(1)
				pchan <- [2]*Particle{p, px}
			}
		}
		wg.Wait()
		close(pchan)

		for _, p := range o.particles {
			p.move(o.trailLength)
		}

		// Check for collisions
		for i, p := range o.particles {
			for _, px := range o.particles[i+1:] {
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
		particles:   []*Particle{},
		trailLength: 20,
		looptime:    5 * time.Millisecond,

		q: make(chan bool),
		c: make(chan command, 20),
	}

	go o.loop()

	return o
}
