all: universe
	./universe

universe: universe.go camera.go drawing.go orrery/orrery.go
	go build
