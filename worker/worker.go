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
	h := request.EndY - request.StartY
	w := request.EndX - request.StartX
	world := stubs.Decode(request.BitMap, h, w)
	mu.Lock()
	globalTurn = 0
	globalWorld = world
	mu.Unlock()

	world = calculateNextState(request.StartY, request.EndY, request.StartX, request.EndX, request.H, world)
	mu.Lock()
	globalWorld = world
	mu.Unlock()

	response.BitMap = stubs.Encode(world, h, w)
	response.StartY = request.StartY
	response.EndY = request.EndY
	return
}

// func (Game GameOfLife) Pauser(request broker.PauserRequest, response *broker.PauserResponse) (err error) {
// 	mu.Lock()
// 	pausing = !pausing
// 	response.CompletedTurns = globalTurn
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

	// for pausing && !quitting {
	// }
	return newWorld
}

func main() {
	pAddr := flag.String("port", "8031", "Port the worker listens on")
	brokerAddr := flag.String("broker", "127.0.0.1", "IP address of the broker")
	remote := flag.String("remote", "1", "Is it running on a local instance?")

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	var listenIP string
	var registerIP string

	if *remote == "0" {

		listenIP = "0.0.0.0"
		imdsHost := "169.254.169.254"
		fmt.Println("Worker being run on remote EC2 instance")
		fmt.Println("Querying IMDS for EC2 public IP address")
		myPrivateIP := stubs.GetMyPrivateIP(imdsHost)
		registerIP = myPrivateIP
	} else {
		listenIP = "127.0.0.1"

		registerIP = "127.0.0.1"

	}

	*brokerAddr = *brokerAddr + ":8030"
	listenAddr := listenIP + ":" + *pAddr
	registerAddr := registerIP + ":" + *pAddr

	fmt.Println("Dialling broker at IP address: ", *brokerAddr)
	client, _ := rpc.Dial("tcp", *brokerAddr)
	defer client.Close()

	rpc.Register(&GameOfLife{})
	listener, _ := net.Listen("tcp", listenAddr)

	request := stubs.WorkerConnectionRequest{Address: registerAddr}
	response := new(stubs.WorkerConnectionResponse)

	client.Call("Broker.RegisterWorker", request, response)

	fmt.Println("Worker listening on", listenAddr)
	rpc.Accept(listener)

}
