all: universe
	./universe

universe: universe.go camera.go draw.go
	go build
