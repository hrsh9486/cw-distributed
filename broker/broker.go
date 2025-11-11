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

func (broker Broker) ScheduleWork(request *stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	// Split up world, call worker methods, recollect
	sectionHeight := request.H / len(globalWorkers)

	mu.Lock()
	globalWorld = request.World
	mu.Unlock()

	var wg sync.WaitGroup

	for turn := 0; turn < request.Turns; turn++ {
		newWorld := make([][]uint8, len(request.World))
		responses := make([]*stubs.BrokerResponse, len(globalWorkers))
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
				responses[i] = new(stubs.BrokerResponse)
				client.Call(loop, &req, &responses[i])

			}(i, upperBound)

		}
		wg.Wait()

		mu.Lock()
		// Need to fix this section to ensure that they are appending it correctly
		for i := range globalWorkers {
			res := responses[i]
			for row := res.StartY; row < res.EndY; row++ {
				newWorld[row] = res.World[row-res.StartY]
			}
		}

		globalWorld = newWorld
		globalCompletedTurns += 1
		globalAliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, newWorld)
		mu.Unlock()
	}

	mu.Lock()
	if request.Turns == 0 {
		response.CompletedTurns = 0
		response.World = request.World
		response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, request.World)
	} else {
		response.CompletedTurns = globalCompletedTurns
		response.World = globalWorld
		response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	}
	mu.Unlock()
	return
}

func (broker Broker) TickerService(request stubs.TickerRequest, response *stubs.TickerResponse) (err error) {
	mu.Lock()
	// fmt.Println(Game.world)
	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
	response.CompletedTurns = globalCompletedTurns
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
	pAddr := flag.String("port", "8030", "The port the server is listening on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&Broker{})

	listener, _ := net.Listen("tcp", ":"+*pAddr)
	fmt.Println("Broker listening on port:", *pAddr)
	rpc.Accept(listener)
	listener.Close()
}
