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
	world := stubs.Decode(request.BitMap, request.SectionHeight, request.SectionWidth)

	world = calculateNextState(request.SectionHeight-2, world)
	response.BitMap = stubs.Encode(world, request.EndY-request.StartY, request.EndX-request.StartX)
	return

}

func worker(sectionHeight int, world [][]uint8, outputChan chan [][]uint8) {
	outputChan <- calculateNextState(sectionHeight, world)
}

// Take a broker state and iteratively calculate the next state for a section of the board
func calculateNextState(sectionHeight int, world [][]uint8) [][]uint8 {
	w := len(world[0])
	newWorld := make([][]uint8, sectionHeight)

	// Populate outer slice, with empty inner slices.
	for i := range newWorld {
		newWorld[i] = make([]uint8, w)
	}

	for y := 1; y < sectionHeight+1; y++ {
		for x := 0; x < w; x++ {
			// Check how many of the current cell's neighbours are alive
			var leftCol int
			var rightCol int

			switch x {
			case 0:
				leftCol = w - 1
				rightCol = x + 1
			case w - 1:
				leftCol = x - 1
				rightCol = 0
			default:
				leftCol = x - 1
				rightCol = x + 1
			}

			currentCell := world[y][x]
			neighboursAlive := (world[y-1][leftCol] / 255) +
				(world[y-1][x] / 255) +
				(world[y-1][rightCol] / 255) +
				(world[y][leftCol] / 255) +
				(world[y][rightCol] / 255) +
				(world[y+1][leftCol] / 255) +
				(world[y+1][x] / 255) +
				(world[y+1][rightCol] / 255)

			// Logic for current cell
			var alive uint8 = 255
			var dead uint8 = 0

			if currentCell == alive {
				if neighboursAlive < 2 {
					newWorld[y-1][x] = dead
				} else if neighboursAlive > 3 {
					newWorld[y-1][x] = dead
				} else if neighboursAlive == 2 || neighboursAlive == 3 {
					newWorld[y-1][x] = currentCell

				}
			} else {
				if neighboursAlive == 3 {
					newWorld[y-1][x] = alive
				} else {
					newWorld[y-1][x] = currentCell
				}
			}
		}
	}

	for pausing && !quitting {
	}
	return newWorld
}

func main() {
	pAddr := flag.String("port", "8031", "Port the worker listens on")
	brokerAddr := flag.String("broker", "127.0.0.1", "IP address of the broker")
	remote := flag.String("remote", "0", "Is it running on a local instance?")

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	var listenIP string
	var registerIP string

	if *remote == "0" {

		listenIP = "0.0.0.0"
		imdsHost := "169.254.169.254"
		fmt.Println("Worker being run on remote EC2 instance")
		fmt.Println("Querying IMDS for EC2 public IP address")
		myPrivateIP := stubs.GetMyIP(imdsHost, true)
		registerIP = myPrivateIP
	} else {
		listenIP = "127.0.0.1"

		registerIP = "127.0.0.1"

	}

	*brokerAddr = *brokerAddr + ":8030"
	listenAddr := listenIP + ":" + *pAddr
	registerAddr := registerIP + ":" + *pAddr

	fmt.Println("Dialling broker at IP address: ", *brokerAddr)
	client, err := rpc.Dial("tcp", *brokerAddr)
	if err != nil {
		fmt.Println("Failed to connect to broker: ", err)
	}
	defer client.Close()

	rpc.Register(&GameOfLife{})
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println(err)
	}

	request := stubs.WorkerConnectionRequest{Address: registerAddr}
	response := new(stubs.WorkerConnectionResponse)

	client.Call("Broker.RegisterWorker", request, response)

	fmt.Println("Worker listening on", listenAddr)
	rpc.Accept(listener)

}
