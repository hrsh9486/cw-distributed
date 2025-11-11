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
	sectionHeight := request.H / len(globalWorkers)
	mu.Lock()
	globalWorld = request.World
	mu.Unlock()

	// Issue is within this for loop, essentially we need to synchronise the workers in some way and ensure packets are being sent back before we can do the next turn.
	for turn := 0; turn < request.Turns; turn++ {
		newWorld := make([][]uint8, len(request.World))
		responses := make([]stubs.BrokerResponse, len(globalWorkers))
		var wg sync.WaitGroup
		for i := range globalWorkers {
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
				req := stubs.BrokerRequest{Turns: request.Turns, StartY: i * sectionHeight, EndY: upperBound, StartX: request.StartX, EndX: request.EndX, H: request.H, World: globalWorld}
				responses[i] = *new(stubs.BrokerResponse)
				client.Call(loop, req, responses[i])
				// Ok, so the call is fine, it enters the worker method, and the worker method executes correctly. However, for some reason, the response is not being recorded properly.
				fmt.Println(responses[i].World)

			}(i, upperBound)

		}
		wg.Wait()

		mu.Lock()
		for i := range globalWorkers {
			fmt.Println(responses[i].World)
			fmt.Println(responses[i].AliveCells)
			fmt.Println(responses[i].CompletedTurns)
		}

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
