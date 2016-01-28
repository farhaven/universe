all: tests universe run

run: universe
	./universe

universe: universe.go camera.go drawing.go orrery/orrery.go vector/vector.go
	go build

tests:
	go test . ./vector ./orrery
