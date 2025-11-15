package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// Functions to be called by the broker to access worker methods
var loop = "GameOfLife.Loop"
var pulse = "GameOfLife.Pulse"

// var saver = "GameOfLife.Saver"
// var quitter = "GameOfLife.Quitter"
// var pauser = "GameOfLife.Pauser"
// var killer = "GameOfLife.Killer"

var mu sync.Mutex

type Broker struct {
}

type worker struct {
	address string
	alive   bool
}

// var globalWorkers []worker
var globalWorkers []string

// var globalWorkers = make(map[string]bool)
var globalWorld [][]uint8
var globalCompletedTurns int
var globalAliveCells []util.Cell
var globalThreads int
var quitting bool

func (broker Broker) RegisterWorker(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	mu.Lock()
	// globalWorkers[request.Address] = true
	globalWorkers = append(globalWorkers, request.Address)
	mu.Unlock()
	return

}

func (broker Broker) ScheduleWork(request *stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	// Split up world, call worker methods, recollect
	// fmt.Println("request start: ", request.StartY, request.EndY, request.StartX, request.EndX)
	// fmt.Println(request.)
	fullWorldHeight := request.EndY - request.StartY
	fullWorldWidth := request.EndX - request.StartX
	sectionHeight := fullWorldHeight / len(globalWorkers)
	mu.Lock()
	globalCompletedTurns = 0
	globalWorld = stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth)
	globalAliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, globalWorld)
	quitting = false
	mu.Unlock()

	var wg sync.WaitGroup

	// Two Strategies:
	// If fail in connecting, then just remove from global workers and distribute work
	// If fail in receving processed, then discard current turn, roll over to previous and start from above

	turn := 0
	for turn < request.Turns && !quitting {
		fmt.Println("Turn:", turn)
		newWorld := make([][]uint8, fullWorldHeight)
		// brokerResponses := make([]*stubs.BrokerResponse, len(globalWorkers))
		numberOfWorkers := len(globalWorkers)

		pulseResponse := make([]stubs.PulseResponse, numberOfWorkers)
		// fmt.Println("globalWorkers", globalWorkers, "len:", numberOfWorkers)
		// workerNumber := 0

		// This is the first part of fault tolerance, see if the workers are alive
		// If the workers are alive, we add them to our map
		// Connect to one client, and see if they're alive
		aliveWorkers := make([]string, 0, len(globalWorkers))
		for i := range globalWorkers {
			fmt.Println("running", globalWorkers)
			// Create a pulse stub and response
			pulseReq := stubs.PulseRequest{Alive: true}
			pulseRes := pulseResponse[i]

			client, err := rpc.Dial("tcp", globalWorkers[i])
			if err != nil {
				fmt.Println("Worker found dead", globalWorkers[i])
				// aliveWorkerCount = aliveWorkerCount - 1
				// YOU NEED TO CONTINUE NO MATTER WHAT
				continue
			}

			defer client.Close()
			// Now that we know what workers are alive, we can divide work up based on those clients

			// Send pulse to client
			err = client.Call(pulse, pulseReq, pulseRes.Alive)
			if err == nil {
				aliveWorkers = append(aliveWorkers, globalWorkers[i])
			}
			// I'm not sure if the following code actually does something but will see
			// They should be alive if you recieve something, so far we recieve false
			if pulseRes.Alive != false {
				fmt.Println("Something failed")
				continue
			}

		}
		globalWorkers = aliveWorkers
		numberOfWorkers = len(globalWorkers)
		brokerResponses := make([]*stubs.BrokerResponse, numberOfWorkers)
		// fmt.Println("globalWorkers", globalWorkers, "len:", numberOfWorkers)
		// Now we actuall run game of life
		for i := range globalWorkers {
			// Now we actually process Game of Life
			var upperBound int
			// Create the upper bounds based on the number of workers
			// fmt.Println("workerNumber", workerNumber)
			if i == numberOfWorkers-1 {
				upperBound = fullWorldHeight
			} else {
				upperBound = (i + 1) * sectionHeight
			}

			// We're going to create a WaitGroup and launch threads for all our workers
			// All the threads need to finish before we can process it
			// This is second part of fault tolerance, if a thread returns nil or error, then we need to
			// handle it with continue, and restart from a previous state
			// This means, skip all of the processing, reset turn and run the pulses
			// This allows us to divide up work from a new set of defintly alive workers
			wg.Add(1)
			// fmt.Println("workerNumber", workerNumber)
			go func(i, upperBound int) {
				defer wg.Done()
				client, err := rpc.Dial("tcp", globalWorkers[i])
				if err != nil {
					fmt.Println("Here is the err:::")
					fmt.Println(err)
				}
				defer client.Close()
				req := stubs.BrokerRequest{
					StartY:          i * sectionHeight,
					EndY:            upperBound,
					StartX:          request.StartX,
					EndX:            request.EndX,
					FullWorldHeight: fullWorldHeight,
					FullWorldWidth:  fullWorldHeight,
					Threads:         globalThreads,
					BitMap:          stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth),
				}
				// fmt.Println("request: ", req.StartY, req.EndY, req.StartX, req.EndX, req.FullWorldHeight, req.FullWorldWidth)
				brokerResponses[i] = new(stubs.BrokerResponse)
				client.Call(loop, &req, &brokerResponses[i])
				// fmt.Println("hi")
			}(i, upperBound)
			// Increment the worker number to access the response array
			// workerNumber = workerNumber + 1
		}
		wg.Wait()

		mu.Lock()
		// fmt.Println("brokerResponse", brokerResponses)
		// workerNumber = 0
		for i := range globalWorkers {
			fmt.Println("this is the address", globalWorkers[i])
			var upperBound int
			if i == len(globalWorkers)-1 {
				upperBound = len(globalWorld)
			} else {
				upperBound = (i + 1) * sectionHeight
			}
			res := brokerResponses[i]
			resWorld := stubs.Decode(res.BitMap, upperBound-(i*sectionHeight), fullWorldWidth)
			for row := i * sectionHeight; row < upperBound; row++ {
				newWorld[row] = resWorld[row-(i*sectionHeight)]
			}
		}

		globalWorld = newWorld
		// fmt.Println("globalWorld:", globalWorld)
		globalCompletedTurns += 1
		globalAliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, newWorld)
		mu.Unlock()
		turn++

		mu.Lock()
		if request.Turns == 0 {
			// fmt.Println("here")
			response.CompletedTurns = 0
			response.BitMap = request.BitMap
			response.AliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth))
		} else {
			response.CompletedTurns = globalCompletedTurns
			response.BitMap = stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth)
			response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
		}
		// fmt.Println("response.Bitmap", response.BitMap)
		mu.Unlock()
	}
	return
}

func (broker Broker) TickerService(request stubs.TickerRequest, response *stubs.TickerResponse) (err error) {
	mu.Lock()
	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
	response.CompletedTurns = globalCompletedTurns
	mu.Unlock()
	return
}

func (broker Broker) Quitter(request stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	mu.Lock()
	quitting = true
	mu.Unlock()
	mu.Lock()
	response.BitMap = stubs.Encode(globalWorld, request.EndY-request.StartY, request.EndX-request.StartX)
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	response.CompletedTurns = globalCompletedTurns
	mu.Unlock()
	return
}

func (broker Broker) Saver(request stubs.SaverRequest, response *stubs.SaverResponse) (err error) {
	mu.Lock()
	response.CompletedTurns = globalCompletedTurns
	response.BitMap = stubs.Encode(globalWorld, len(globalWorld), len(globalWorld[0]))
	mu.Unlock()
	return
}

func getAliveCells(h, w int, world [][]uint8) []util.Cell {
	var aliveCells []util.Cell
	var alive uint8 = 255
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if world[y][x] == alive {
				aliveCell := util.Cell{X: x, Y: y}
				aliveCells = append(aliveCells, aliveCell)
			}
		}
	}
	return aliveCells
}

// Register workers
// Receive client request
// Schedule work
// Return final result
func main() {
	pAddr := flag.String("port", "8030", "The port the broker is listening on")
	remote := flag.String("remote", "0", "Is it running on a local instance?")
	threads := flag.String("threads", "1", "Specify number of threads to run on each worker")

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	globalThreads, _ = strconv.Atoi(*threads)

	var listenIP string

	if *remote == "0" {
		listenIP = "0.0.0.0"
		imdsHost := "169.254.169.254"
		myPrivateIP := stubs.GetMyIP(imdsHost, true)
		myPublicIP := stubs.GetMyIP(imdsHost, false)
		fmt.Println("Broker running on EC2 instance, with private IP: " + myPrivateIP)
		fmt.Println("Public IP:" + myPublicIP)
		fmt.Println("")
	} else {
		listenIP = "127.0.0.1"

	}

	rpc.Register(&Broker{})

	listener, err := net.Listen("tcp", listenIP+":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Broker listening on port:", listenIP+":"+*pAddr)
	rpc.Accept(listener)
	listener.Close()
}
