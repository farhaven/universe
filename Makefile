all: universe
	./universe

universe: universe.go camera.go draw.go planets.go
	go build
