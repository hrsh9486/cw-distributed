package gol

import (
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

// Functions to be called by the client to access broker methods
var scheduleWork = "Broker.ScheduleWork"
var brokerTickerService = "Broker.TickerService"
var brokerSaver = "Broker.Saver"
var brokerQuitter = "Broker.Quitter"
var brokerPauser = "Broker.Pauser"
var brokerKiller = "Broker.Killer"

type distributorChannels struct {
	events         chan<- Event
	ioCommand      chan<- ioCommand
	ioIdle         <-chan bool
	ioFilename     chan<- string
	ioOutput       chan<- uint8
	ioInput        <-chan uint8
	keyPressesChan <-chan rune
}

func handleEvent(ticker *time.Ticker, done chan bool, client *rpc.Client, c distributorChannels, h int, w int, isPaused bool, fileName string) {
	for {
		select {
		case <-ticker.C:
			request := stubs.TickerRequest{StartY: 0, EndY: h, StartX: 0, EndX: w, H: h}
			response := new(stubs.TickerResponse)
			client.Call(brokerTickerService, request, response)
			c.events <- AliveCellsCount{response.CompletedTurns, response.AliveCellsCount}

		case keyPress := <-c.keyPressesChan:
			switch keyPress {
			case 'q':
				request := stubs.BrokerRequest{StartY: 0, EndY: h, StartX: 0, EndX: w, H: h}
				response := new(stubs.BrokerResponse)
				client.Call(brokerQuitter, request, response)

			case 'p':
				request := stubs.PauserRequest{}
				response := new(stubs.PauserResponse)
				client.Call(brokerPauser, request, response)
				if !isPaused {
					isPaused = true
					c.events <- StateChange{response.CompletedTurns, Paused}
				} else {
					isPaused = false
					c.events <- StateChange{response.CompletedTurns, Executing}
				}

				// Need to add something here to deal with logic on client side
			case 's':
				request := stubs.SaverRequest{}
				response := new(stubs.SaverResponse)
				client.Call(brokerSaver, request, response)
				outputFileName := fileName + "x" + strconv.Itoa(response.CompletedTurns)
				c.ioCommand <- ioOutput
				c.ioFilename <- outputFileName
				// Need to pass in height and width
				responseWorld := stubs.Decode(response.BitMap, h, w)
				for i := range responseWorld {
					for j := 0; j < w; j++ {
						c.ioOutput <- responseWorld[i][j]
					}
				}

				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
				c.events <- ImageOutputComplete{response.CompletedTurns, outputFileName}

				// Need to add something here to deal with logic on client side
			case 'k':
				request := stubs.BrokerRequest{StartY: 0, EndY: h, StartX: 0, EndX: w, H: h}
				response := new(stubs.BrokerResponse)
				client.Call(brokerKiller, request, response)
			}
		case <-done:
			return
		}
	}
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

	done := make(chan bool)
	ticker := time.NewTicker(2 * time.Second)

	c.events <- StateChange{0, Executing}
	// TODO: Execute all turns of the Game of Life.
	// broker := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	// flag.Parse()
	broker := "3.237.18.107"
	// broker = "127.0.0.1"
	port := "8030"
	broker = broker + ":" + port

	//TODO: connect to the RPC server and send the request(s)
	client, err := rpc.Dial("tcp", broker)
	if err != nil {
		fmt.Println("Failed to connect to broker: ", err)
	}
	defer client.Close()

	request := stubs.ClientRequest{
		Turns:  p.Turns,
		StartY: 0,
		EndY:   h,
		StartX: 0,
		EndX:   w,
		H:      h,
		BitMap: stubs.Encode(world, h, w)}
	response := new(stubs.ClientResponse)

	go handleEvent(ticker, done, client, c, h, w, false, fileName)
	client.Call(scheduleWork, request, response)
	world = stubs.Decode(response.BitMap, w, h)

	c.events <- FinalTurnComplete{response.CompletedTurns, response.AliveCells}

	outputFileName := fileName + "x" + strconv.Itoa(response.CompletedTurns)
	c.ioCommand <- ioOutput
	c.ioFilename <- outputFileName
	for i := range world {
		for j := 0; j < w; j++ {
			c.ioOutput <- world[i][j]
		}
	}

	ticker.Stop()
	done <- true
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- ImageOutputComplete{response.CompletedTurns, outputFileName}
	c.events <- StateChange{response.CompletedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
