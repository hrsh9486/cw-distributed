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

// var globalWorkers []string
var globalWorkers = make(map[string]bool)
var globalWorld [][]uint8
var globalCompletedTurns int
var globalAliveCells []util.Cell
var globalThreads int
var quitting bool

func (broker Broker) RegisterWorker(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	mu.Lock()
	globalWorkers[request.Address] = true
	// globalWorkers = append(globalWorkers, request.Address)
	mu.Unlock()
	return

}

func (broker Broker) ScheduleWork(request *stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	// Split up world, call worker methods, recollect

	fullWorldHeight := request.EndY - request.StartY
	fullWorldWidth := request.EndX - request.StartX
	// sectionHeight := fullWorldHeight / len(globalWorkers)
	mu.Lock()
	globalCompletedTurns = 0
	globalWorld = stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth)
	globalAliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, globalWorld)
	quitting = false
	mu.Unlock()

	// var wg sync.WaitGroup

	turn := 0
	for turn < request.Turns && !quitting {
		// newWorld := make([][]uint8, fullWorldHeight)
		// responses := make([]*stubs.BrokerResponse, len(globalWorkers))
		// received := make([]int, len(globalWorkers))
		numberOfWorkers := len(globalWorkers)

		pulseResponse := make([]stubs.PulseResponse, numberOfWorkers)
		fmt.Println("globalWorkers", globalWorkers, "len:", numberOfWorkers)
		workerNumber := 0
		for address := range globalWorkers {
			// Create a pulse stub and response
			pulseReq := stubs.PulseRequest{Alive: true}
			pulseRes := pulseResponse[workerNumber]

			// Connect to one client, and see if they're alive
			client, err := rpc.Dial("tcp", address)
			if err != nil {
				delete(globalWorkers, address)
				// YOU NEED TO CONTINUE NO MATTER WHAT
				continue
			}
			defer client.Close()

			// Send pulse to client
			client.Call(pulse, pulseReq, pulseRes)
			fmt.Println("resAlive:", pulseRes.Alive)
			// I'm not sure if the following code actually does something but will see
			// They should be alive if you recieve something, so far we recieve false
			if pulseRes.Alive != false {
				fmt.Println("Something failed")
				continue
			}

			// Now we actually send the game of life over

			// Increment the worker number to access the response array
			workerNumber = workerNumber + 1
		}
	}

	// 	fmt.Println("recieved", i)
	// 	var upperBound int
	// 	if i == len(globalWorkers)-1 {
	// 		upperBound = len(globalWorld)
	// 	} else {
	// 		upperBound = (i + 1) * sectionHeight
	// 	}

	// 	wg.Add(1)
	// 	go func(i, upperBound int) {
	// 		defer wg.Done()
	// 		client, err := rpc.Dial("tcp", globalWorkers[i])
	// 		if err != nil {
	// 			fmt.Println("Here is the err:::")
	// 			fmt.Println(err)
	// 		}
	// 		defer client.Close()
	// 		req := stubs.BrokerRequest{
	// 			StartY:          i * sectionHeight,
	// 			EndY:            upperBound,
	// 			StartX:          request.StartX,
	// 			EndX:            request.EndX,
	// 			FullWorldHeight: fullWorldHeight,
	// 			FullWorldWidth:  fullWorldHeight,i
	// 			Threads:         globalThreads,
	// 			BitMap:          stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth)}

	// 		responses[i] = new(stubs.BrokerResponse)
	// 		client.Call(loop, &req, &responses[i])
	// 		fmt.Println("bitmap:", responses[i].BitMap)
	// 		if responses[i] == nil {
	// 			for {
	// 				fmt.Println("something died")
	// 			}
	// 		}
	// 		if responses[i] == nil {
	// 			for {
	// 				fmt.Println("something died")
	// 			}
	// 		}
	// 	}(i, upperBound)

	// }

	// for i := range responses {
	// 	if responses[i] == nil {
	// 		for {
	// 			fmt.Println("something died")
	// 		}
	// 	}
	// }
	// wg.Wait()

	// mu.Lock()
	// // Need to fix this section to ensure that they are appending it correctly
	// for i := range globalWorkers {
	// 	var upperBound int
	// 	if i == len(globalWorkers)-1 {
	// 		upperBound = len(globalWorld)
	// 	} else {
	// 		upperBound = (i + 1) * sectionHeight
	// 	}
	// 	res := responses[i]
	// 	resWorld := stubs.Decode(res.BitMap, upperBound-(i*sectionHeight), fullWorldWidth)
	// 	for row := i * sectionHeight; row < upperBound; row++ {
	// 		newWorld[row] = resWorld[row-(i*sectionHeight)]
	// 	}
	// }

	// globalWorld = newWorld
	// globalCompletedTurns += 1
	// globalAliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, newWorld)
	// mu.Unlock()
	// turn++

	// mu.Lock()
	// if request.Turns == 0 {
	// 	response.CompletedTurns = 0
	// 	response.BitMap = request.BitMap
	// 	response.AliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth))
	// } else {
	// 	response.CompletedTurns = globalCompletedTurns
	// 	response.BitMap = stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth)
	// 	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	// }
	// mu.Unlock()
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
