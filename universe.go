package main

import (
	"log"
	"os"
	"runtime/pprof"

	"git.c3pb.de/farhaven/universe/orrery"
	"git.c3pb.de/farhaven/universe/ui"
)

func main() {
	f, err := os.Create("cpuprofile.prof")
	if err != nil {
		log.Printf(`can't create CPU profile: %s`, err)
	}
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	o := orrery.New()

	width, height := 1024, 768
	ctx := ui.NewDrawContext(width, height, o)

	log.Println(`waiting for ui to shut down`)
	ctx.WaitForShutdown()
}
