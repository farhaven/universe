package ui

import (
	"log"
	"os"
	"testing"

	"git.c3pb.de/farhaven/universe/vector"
	"github.com/go-gl/gl/v2.1/gl"
)

func TestMain(m *testing.M) {
	if err := gl.Init(); err != nil {
		log.Fatalf(`can't init GL: %s`, err)
	}
	os.Exit(m.Run())
}

func TestSphereInFrustum(t *testing.T) {
	c := NewCamera(1440, 900, -40, 40, 10)
	c.Update()

	if f := c.SphereInFrustum(vector.V3{0, 0, 0}, 30); f != INTERSECT {
		t.Errorf(`expected INTERSECT, got %s`, f.String())
	}

	if f := c.SphereInFrustum(vector.V3{0, 0, -100}, 1); f != OUTSIDE {
		t.Errorf(`expected OUTSIDE, got %s`, f.String())
	}
}
