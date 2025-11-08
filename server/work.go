package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/gol"
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
func (Game GameOfLife) Loop(request gol.WorkerRequest, response *gol.WorkerResponse) (err error) {
	quitting = false
	pausing = false
	isKilled = false
	turn := 0
	world := request.World
	mu.Lock()
	globalTurn = 0
	globalWorld = world
	mu.Unlock()
	// fmt.Println("never run twice")
	for turn < request.Turns && !quitting && !isKilled {
		// rishi for turn < request.Turns && !quitting {
		// fmt.Println("isKilled init", isKilled)
		for pausing && !quitting {
			// fmt.Println("isKilled within", isKilled)
			// if isKilled {
			// 	fmt.Println()
			// 	mu.Lock()
			// 	isKilled = true
			// 	mu.Unlock()
			// 	break
			// }
			fmt.Println("killed?", isKilled)
			mu.Lock()
			if isKilled {
				fmt.Println("killed")
			}
			mu.Unlock()
			// fmt.Println("")
			fmt.Println("oops")
		}
		fmt.Println("beep:", isKilled)
		if isKilled {
			os.Exit(0)
		}
		fmt.Println("last seen here")
		mu.Lock()

		world = calculateNextState(request.StartY, request.EndY, request.StartX, request.EndX, request.H, world)

		// mu.Unlock()

		turn += 1
		// mu.Lock()
		globalTurn = turn
		globalWorld = world
		// mu.Unlock()
		response.CompletedTurns = turn
		mu.Unlock()
	}

	response.World = world
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, world)
	return
}

func (Game GameOfLife) TickerService(request gol.TickerRequest, response *gol.TickerResponse) (err error) {
	mu.Lock()
	// fmt.Println(Game.world)
	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
	response.CompletedTurns = globalTurn
	mu.Unlock()
	return

}

func (Game GameOfLife) Quitter(request gol.WorkerRequest, response *gol.WorkerResponse) (err error) {
	mu.Lock()
	quitting = true
	// mu.Unlock()
	// mu.Lock()
	response.World = globalWorld
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	response.CompletedTurns = globalTurn
	mu.Unlock()
	return
}

func (Game GameOfLife) Pauser(request gol.PauserRequest, response *gol.PauserResponse) (err error) {
	mu.Lock()
	pausing = !pausing
	response.CompletedTurns = globalTurn
	mu.Unlock()
	fmt.Println("Permanantly locked")
	return
}

func (Game GameOfLife) Saver(request gol.SaverRequest, response *gol.SaverResponse) (err error) {
	mu.Lock()
	response.CompletedTurns = globalTurn
	response.World = globalWorld
	mu.Unlock()
	return
}

func (Game GameOfLife) Killer(request gol.KillerRequest, response *gol.KillerResponse) (err error) {
	mu.Lock()
	isKilled = true
	fmt.Println("inside lock", isKilled)
	mu.Unlock()
	fmt.Println("outside lock", isKilled)
	if !pausing {
		// fmt.Println("wrong")
		os.Exit(0)
	}
	// os.Exit(0)
	return
}

// Take a GoL state and iteratively calculate the next state for a section of the board
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

	// for pausing && !quitting {
	// 	fmt.Println("oops")
	// }
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

	fmt.Println("Listening for a connection...")
	pAddr := flag.String("port", "8030", "The port the server is listening on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOfLife{})

	listener, _ := net.Listen("tcp", ":"+*pAddr)
	for !isKilled {
		rpc.Accept(listener)
	}
	listener.Close()
}
