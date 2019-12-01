package ui

import (
	"log"
	"os"
	"testing"

	"git.c3pb.de/farhaven/universe/vector"
	"github.com/go-gl/gl/v2.1/gl"
)

func TestMain(m *testing.M) {
	err := gl.Init()
	if err != nil {
		log.Fatalf(`can't init GL: %s`, err)
	}
	os.Exit(m.Run())
}

func TestSphereInFrustum(t *testing.T) {
	c := NewCamera(1440, 900, -40, 40, 10)
	c.Update()

	f := c.SphereInFrustum(vector.V3{X: 0, Y: 0, Z: 0}, 30)
	if f != INTERSECT {
		t.Errorf(`expected INTERSECT, got %s`, f.String())
	}

	f = c.SphereInFrustum(vector.V3{X: 0, Y: 0, Z: -100}, 1)
	if f != OUTSIDE {
		t.Errorf(`expected OUTSIDE, got %s`, f.String())
	}
}
