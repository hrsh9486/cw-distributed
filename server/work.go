package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
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
var mu sync.Mutex

// Calculate a certain number of game of life states
func (Game GameOfLife) Loop(request gol.WorkerRequest, response *gol.WorkerResponse) (err error) {
	turn := 0
	world := request.World
	mu.Lock()
	globalTurn = 0
	globalWorld = world
	mu.Unlock()

	for turn < request.Turns {
		world = calculateNextState(request.StartY, request.EndY, request.StartX, request.EndX, request.H, world)
		turn += 1
		mu.Lock()
		globalTurn = turn
		globalWorld = world
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
	defer listener.Close()
	rpc.Accept(listener)
}
