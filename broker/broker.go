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

// var saver = "GameOfLife.Saver"
// var quitter = "GameOfLife.Quitter"
// var pauser = "GameOfLife.Pauser"
// var killer = "GameOfLife.Killer"

var mu sync.Mutex

type Broker struct {
}

var globalWorkers []string
var globalWorld [][]uint8
var globalCompletedTurns int
var globalAliveCells []util.Cell
var quitting bool
var globalThreads int

func (broker Broker) RegisterWorker(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	mu.Lock()
	globalWorkers = append(globalWorkers, request.Address)
	mu.Unlock()
	fmt.Println("Worker Registered at IP:" + request.Address)
	return

}

func (broker Broker) ScheduleWork(request *stubs.ClientRequest, response *stubs.ClientResponse) (err error) {

	//Preamble
	//------------------------------------------------------------------
	// Initialise variables to pass into workers
	sectionHeight := request.H / len(globalWorkers)
	h := request.EndY - request.StartY
	w := request.EndX - request.StartX
	var wg sync.WaitGroup
	turn := 0
	newWorld := make([][]uint8, h)
	responses := make([]*stubs.BrokerResponse, len(globalWorkers))

	// Initialise global variables for tracking state
	mu.Lock()
	globalCompletedTurns = 0
	globalWorld = stubs.Decode(request.BitMap, h, w)
	quitting = false
	mu.Unlock()
	//------------------------------------------------------------------

	// Iterate over workers, call worker methods
	for i := range globalWorkers {
		// Calculate which workers are neighbours with each other to facilitate halo exchange.
		var neighbours []string
		if len(globalWorkers) == 1 {
			neighbours = make([]string, 0)

		} else if len(globalWorkers) == 2 {
			neighbours = make([]string, 1)
			neighbours = append(neighbours, globalWorkers[(i+1)%2])

		} else {
			neighbours = make([]string, 2)
			next := globalWorkers[(i+1)%len(globalWorkers)]
			var prev string
			if i == (0) {
				prev = globalWorkers[len(globalWorkers)-1]
			} else {
				prev = globalWorkers[i-1]
			}
			neighbours = append(neighbours, prev, next)

		}
		var upperBound int
		if i == len(globalWorkers)-1 {
			upperBound = request.H
		} else {
			upperBound = (i + 1) * sectionHeight
		}

		wg.Add(1)
		go func(i, upperBound int) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", globalWorkers[i])
			if err != nil {
				fmt.Println(err)
			}
			defer client.Close()
			req := stubs.BrokerRequest{
				Turns:      request.Turns,
				StartY:     i * sectionHeight,
				EndY:       upperBound,
				StartX:     request.StartX,
				EndX:       request.EndX,
				H:          request.H,
				BitMap:     stubs.Encode(globalWorld, h, w),
				Threads:    globalThreads,
				Neighbours: neighbours}
			responses[i] = new(stubs.BrokerResponse)
			client.Call(loop, &req, &responses[i])
		}(i, upperBound)

	}
	wg.Wait()

	mu.Lock()
	// Need to fix this section to ensure that they are appending it correctly
	for i := range globalWorkers {
		res := responses[i]
		h := res.EndY - res.StartY
		w := request.EndX - request.StartX
		workerWorld := stubs.Decode(res.BitMap, h, w)
		for row := res.StartY; row < res.EndY; row++ {
			newWorld[row] = workerWorld[row-res.StartY]
		}

		globalWorld = newWorld
		globalAliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, newWorld)
		globalCompletedTurns = turn
		mu.Unlock()
		turn++
	}

	mu.Lock()
	if request.Turns == 0 {
		response.CompletedTurns = 0
		response.BitMap = request.BitMap
		response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, stubs.Decode(request.BitMap, h, w))
	} else {
		response.CompletedTurns = globalCompletedTurns
		response.BitMap = stubs.Encode(globalWorld, h, w)
		response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	}
	mu.Unlock()
	return
}

func (broker Broker) TickerService(request stubs.TickerRequest, response *stubs.TickerResponse) (err error) {
	mu.Lock()
	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
	response.CompletedTurns = globalCompletedTurns + 1
	mu.Unlock()
	return
}

func (broker Broker) Quitter(request stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	mu.Lock()
	quitting = true
	mu.Unlock()
	mu.Lock()
	h := request.EndY - request.StartY
	w := request.EndX - request.StartX
	response.BitMap = stubs.Encode(globalWorld, h, w)
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	response.CompletedTurns = globalCompletedTurns
	mu.Unlock()
	return
}

func (broker Broker) Saver(request stubs.SaverRequest, response *stubs.SaverResponse) (err error) {
	mu.Lock()
	response.CompletedTurns = globalCompletedTurns

	h := len(globalWorld)
	w := len(globalWorld[0])
	response.BitMap = stubs.Encode(globalWorld, h, w)
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
