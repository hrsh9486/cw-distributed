package gol

import (
	"net/rpc"
	"strconv"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// TODO: Create a 2D slice to store the world.
	// Preamble
	// ---------------------------------------------------------------------
	// Extract information from parameters
	h := p.ImageHeight
	w := p.ImageWidth
	fileName := strconv.Itoa(h) + "x" + strconv.Itoa(w)
	outputFileName := fileName + "x" + strconv.Itoa(p.Turns)

	// Initialise 2D slice to store world
	world := make([][]uint8, h)
	for i := range world {
		world[i] = make([]uint8, w)
	}
	// Construct deep copy of world, using data passed through IO channels.
	c.ioCommand <- ioInput
	c.ioFilename <- fileName

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			val := <-c.ioInput
			world[y][x] = val
		}

	}

	turn := 0
	c.events <- StateChange{0, Executing}
	// TODO: Execute all turns of the Game of Life.
	// server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	// flag.Parse()
	server := "127.0.0.1:8030"

	//TODO: connect to the RPC server and send the request(s)
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	request := Request{Turns: p.Turns, StartY: 0, EndY: h, StartX: 0, EndX: w, H: h, World: world}
	response := new(Response)
	client.Call(loop, request, response)
	world = response.World

	c.ioCommand <- ioOutput
	c.ioFilename <- outputFileName
	for i := range world {
		for j := 0; j < w; j++ {
			c.ioOutput <- world[i][j]
		}
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, response.AliveCells}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
