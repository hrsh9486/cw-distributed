package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// Functions to be called by the broker to access worker methods
var loop = "GameOfLife.Loop"

// var tickerService = "GameOfLife.TickerService"
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

func (broker Broker) RegisterWorker(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	mu.Lock()
	globalWorkers = append(globalWorkers, request.Address)
	mu.Unlock()
	return

}

func (broker Broker) ScheduleWork(request stubs.BrokerRequest, response *stubs.BrokerResponse) (err error) {
	// Split up world, call worker methods, recollect
	numWorkers := len(globalWorkers)
	sectionHeight := request.H / numWorkers
	var responses []stubs.BrokerResponse
	mu.Lock()
	globalWorld = request.World
	mu.Unlock()

	for turn := 0; turn < request.Turns; turn++ {
		for i := 0; i < numWorkers; i++ {
			var upperBound int
			if i == len(globalWorkers)-1 {
				upperBound = request.H
			} else {
				upperBound = (i + 1) * sectionHeight
			}
			client, _ := rpc.Dial("tcp", globalWorkers[i])
			request := stubs.BrokerRequest{Turns: request.Turns, StartY: i * sectionHeight, EndY: upperBound, StartX: request.StartX, EndX: request.EndX, H: request.H, World: globalWorld}
			responses = append(responses, *new(stubs.BrokerResponse))
			client.Call(loop, request, responses[i])
		}

		var newWorld [][]uint8
		for i := 0; i < numWorkers; i++ {
			mu.Lock()
			newWorld = append(newWorld, responses[i].World...)
			globalAliveCells = append(globalAliveCells, responses[i].AliveCells...)
			mu.Unlock()
		}
		mu.Lock()
		globalWorld = newWorld
		globalCompletedTurns += 1
		mu.Unlock()

	}

	response.CompletedTurns = globalCompletedTurns
	response.World = globalWorld
	response.AliveCells = globalAliveCells
	return
}

// Register workers
// Receive client request
// Schedule work
// Return final result
func main() {
	pAddr := flag.String("port", "8030", "The port the server is listening on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&Broker{})

	listener, _ := net.Listen("tcp", ":"+*pAddr)
	fmt.Println("Broker listening on port:", *pAddr)
	rpc.Accept(listener)
	listener.Close()
}
