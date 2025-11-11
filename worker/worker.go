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

// Need a mutex lock on world and turn
type GameOfLife struct {
}

var globalWorld [][]uint8
var globalTurn int
var quitting bool
var pausing bool
var mu sync.Mutex
var isKilled bool

// Calculate a certain number of game of life states
func (Game GameOfLife) Loop(request stubs.BrokerRequest, response *stubs.BrokerResponse) (err error) {
	// quitting = false
	// pausing = false
	// isKilled = false
	// turn := 0
	world := request.World
	mu.Lock()
	globalTurn = 0
	globalWorld = world
	mu.Unlock()

	world = calculateNextState(request.StartY, request.EndY, request.StartX, request.EndX, request.H, world)
	// turn += 1
	mu.Lock()
	// globalTurn = turn
	globalWorld = world
	mu.Unlock()
	// response.CompletedTurns = turn

	response.World = world
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, world)
	return
}

// func (Game GameOfLife) TickerService(request broker.TickerRequest, response *broker.TickerResponse) (err error) {
// 	mu.Lock()
// 	// fmt.Println(Game.world)
// 	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
// 	response.CompletedTurns = globalTurn
// 	mu.Unlock()
// 	return

// }

// func (Game GameOfLife) Quitter(request broker.WorkerRequest, response *broker.WorkerResponse) (err error) {
// 	mu.Lock()
// 	quitting = true
// 	mu.Unlock()
// 	mu.Lock()
// 	response.World = globalWorld
// 	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
// 	response.CompletedTurns = globalTurn
// 	mu.Unlock()
// 	return
// }

// func (Game GameOfLife) Pauser(request broker.PauserRequest, response *broker.PauserResponse) (err error) {
// 	mu.Lock()
// 	pausing = !pausing
// 	response.CompletedTurns = globalTurn
// 	mu.Unlock()
// 	return
// }

// func (Game GameOfLife) Saver(request broker.SaverRequest, response *broker.SaverResponse) (err error) {
// 	mu.Lock()
// 	response.CompletedTurns = globalTurn
// 	response.World = globalWorld
// 	mu.Unlock()
// 	return
// }

// func (Game GameOfLife) Killer(request broker.KillerRequest, response *broker.KillerResponse) (err error) {
// 	isKilled = true
// 	return
// }

// Take a broker state and iteratively calculate the next state for a section of the board
func calculateNextState(startY, endY, startX, endX, h int, world [][]uint8) [][]uint8 {
	w := endX - startX
	newWorld := make([][]uint8, endY-startY)

	// Populate outer slice, with empty inner slices.
	for i := range newWorld {
		newWorld[i] = make([]uint8, w)
	}

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			// Check how many of the current cell's neighbours are alive
			currentCell := world[y][x]
			neighboursAlive := (world[(y+h-1)%h][(x+w-1)%w] / 255) +
				(world[(y+h-1)%h][(x+w)%w] / 255) +
				(world[(y+h-1)%h][(x+w+1)%w] / 255) +
				(world[(y+h)%h][(x+w-1)%w] / 255) +
				(world[(y+h)%h][(x+w+1)%w] / 255) +
				(world[(y+h+1)%h][(x+w-1)%w] / 255) +
				(world[(y+h+1)%h][(x+w)%w] / 255) +
				(world[(y+h+1)%h][(x+w+1)%w] / 255)

			// Logic for current cell
			var alive uint8 = 255
			var dead uint8 = 0
			offsetY := y - startY
			offsetX := x - startX

			if currentCell == alive {
				if neighboursAlive < 2 {
					newWorld[offsetY][offsetX] = dead
				} else if neighboursAlive > 3 {
					newWorld[offsetY][offsetX] = dead
				} else if neighboursAlive == 2 || neighboursAlive == 3 {
					newWorld[offsetY][offsetX] = currentCell

				}
			} else {
				if neighboursAlive == 3 {
					newWorld[offsetY][offsetX] = alive
				} else {
					newWorld[offsetY][offsetX] = currentCell
				}
			}
		}
	}

	for pausing && !quitting {
	}
	return newWorld
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

func main() {
	pAddr := flag.String("port", "8031", "Port the worker listens on")
	workerAddr := flag.String("address", "127.0.0.1", "IP address of worker")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	brokerAddr := "127.0.0.1:8030"
	client, _ := rpc.Dial("tcp", brokerAddr)
	defer client.Close()

	*workerAddr = *workerAddr + ":" + *pAddr
	rpc.Register(&GameOfLife{})
	listener, _ := net.Listen("tcp", *workerAddr)

	request := stubs.WorkerConnectionRequest{Address: *workerAddr}
	response := new(stubs.WorkerConnectionResponse)

	client.Call("Broker.RegisterWorker", request, response)

	fmt.Println("Worker listening on", *workerAddr)
	rpc.Accept(listener)

}
