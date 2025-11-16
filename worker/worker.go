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

type GameOfLife struct {
}

var mu sync.Mutex
var lastCompletedTurn int
var globalWorld [][]uint8
var aboveNeighbour string
var belowNeighbour string

func (Game GameOfLife) SendBottomHalo(request stubs.WorkerHaloRequest, response *stubs.WorkerHaloResponse) (err error) {
	waitForNextTurn(request.LastCompletedTurn)
	mu.Lock()
	response.Row = globalWorld[len(globalWorld)-1]
	mu.Unlock()
	return
}

func (Game GameOfLife) SendTopHalo(request stubs.WorkerHaloRequest, response *stubs.WorkerHaloResponse) (err error) {
	waitForNextTurn(request.LastCompletedTurn)
	mu.Lock()
	response.Row = globalWorld[0]
	mu.Unlock()
	return
}

func (Game GameOfLife) Loop(request stubs.BrokerRequest, response *stubs.BrokerResponse) (err error) {
	// Unpack data from request, initialise worker neighbours
	// threads := request.Threads
	// workerHeight := (request.EndY - request.StartY) / threads
	turn := 0
	mu.Lock()
	globalWorld = stubs.Decode(request.BitMap, request.FullWorldHeight, request.FullWorldWidth)
	lastCompletedTurn = 0
	if len(request.Neighbours) == 1 {
		aboveNeighbour = request.Neighbours[0]
		belowNeighbour = request.Neighbours[0]
	} else if len(request.Neighbours) == 2 {
		aboveNeighbour = request.Neighbours[0]
		belowNeighbour = request.Neighbours[1]
	}
	mu.Unlock()

	// Establish connections with neighbours
	topClient, err := rpc.Dial("tcp", aboveNeighbour)
	if err != nil {
		return err
	}
	defer topClient.Close()
	bottomClient, err := rpc.Dial("tcp", belowNeighbour)
	if err != nil {
		return err
	}
	defer bottomClient.Close()

	// if threads == 1 {
	for turn < request.Turns {
		mu.Lock()
		world := copyWorld(globalWorld)
		mu.Unlock()
		topRequest := stubs.WorkerHaloRequest{LastCompletedTurn: lastCompletedTurn}
		bottomRequest := stubs.WorkerHaloRequest{LastCompletedTurn: lastCompletedTurn}
		topResponse := new(stubs.WorkerHaloResponse)
		bottomResponse := new(stubs.WorkerHaloResponse)
		topClient.Call("GameOfLife.SendBottomHalo", topRequest, topResponse)
		bottomClient.Call("GameOfLife.SendTopHalo", bottomRequest, bottomResponse)
		topRow := topResponse.Row
		bottomRow := bottomResponse.Row
		// Make sure that we can proceed

		updateWorld := calculateNextState(request.StartY, request.EndY, 0, request.FullWorldWidth, world, topRow, bottomRow)

		// Update the global state
		mu.Lock() // Only lock for the brief update of the global state
		globalWorld = updateWorld
		turn++
		lastCompletedTurn = turn // Update completed turn
		mu.Unlock()
	}
	response.BitMap = stubs.Encode(globalWorld, request.EndY-request.StartY, request.EndX-request.StartX)
	response.CompletedTurns = lastCompletedTurn
	return
}

func waitForNextTurn(targetTurn int) {
	for {
		mu.Lock()
		if lastCompletedTurn >= targetTurn {
			mu.Unlock()
			return
		}
		mu.Unlock()
		// Wait briefly before checking again to avoid burning CPU (Spinlock)
		time.Sleep(1 * time.Millisecond)
	}
}

// Helper to copy world slice
func copyWorld(world [][]uint8) [][]uint8 {
	newWorld := make([][]uint8, len(world))
	for i := range world {
		newWorld[i] = make([]uint8, len(world[i]))
		copy(newWorld[i], world[i])
	}
	return newWorld
}

func worker(startY, endY, startX, endX int, world [][]uint8, topRow, bottomRow []uint8, outputChan chan [][]uint8) {
	outputChan <- calculateNextState(startY, endY, startX, endX, world, topRow, bottomRow)
}

func calculateNextState(startY, endY, startX, endX int, world [][]uint8, topRow, bottomRow []uint8) [][]uint8 {
	w := len(world[0])
	h := len(world)
	newWorld := make([][]uint8, endY-startY)

	for i := range newWorld {
		newWorld[i] = make([]uint8, w)
	}

	// Wait to receive data from top and bottom client who will update my globalHalo values
	// Something like (if globalTopHalo is empty, wait, otherwise topRow == globalTopHalo)

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			var neighboursAlive uint8
			// Send my own halo data first, before waiting to receive anything.
			// For the first row of this worker's section
			switch y {
			case startY:
				neighboursAlive = topRow[(x+w-1)%w]/255 +
					topRow[x%w]/255 +
					topRow[(x+1)%w]/255 +
					world[y][(x+w-1)%w]/255 +
					world[y][(x+1)%w]/255 +
					world[(y+1)%h][(x+w-1)%w]/255 +
					world[(y+1)%h][x%w]/255 +
					world[(y+1)%h][(x+1)%w]/255
			case endY - 1:
				neighboursAlive = world[(y-1)%h][(x+w-1)%w]/255 +
					world[(y-1)%h][x%w]/255 +
					world[(y-1)%h][(x+1)%w]/255 +
					world[y][(x+w-1)%w]/255 +
					world[y][(x+1)%w]/255 +
					bottomRow[(x+w-1)%w]/255 +
					bottomRow[x%w]/255 +
					bottomRow[(x+1)%w]/255
			default:
				neighboursAlive = world[(y-1)%h][(x+w-1)%w]/255 +
					world[(y-1)%h][x%w]/255 +
					world[(y-1)%h][(x+1)%w]/255 +
					world[y][(x+w-1)%w]/255 +
					world[y][(x+1)%w]/255 +
					world[(y+1)%h][(x+w-1)%w]/255 +
					world[(y+1)%h][x%w]/255 +
					world[(y+1)%h][(x+1)%w]/255
			}

			// Game of Life rules
			offsetY := y - startY
			offsetX := x - startX
			currentCell := world[y][x]

			if currentCell == 255 {
				// Cell is alive
				if neighboursAlive == 2 || neighboursAlive == 3 {
					newWorld[offsetY][offsetX] = 255
				} else {
					newWorld[offsetY][offsetX] = 0
				}
			} else {
				// Cell is dead
				if neighboursAlive == 3 {
					newWorld[offsetY][offsetX] = 255
				} else {
					newWorld[offsetY][offsetX] = 0
				}
			}
		}

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

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

// }
//  else {
// 	for turn < request.Turns {
// 		mu.Lock()
// 		world := copyWorld(globalWorld)
// 		topRequest := stubs.WorkerHaloRequest{LastCompletedTurn: lastCompletedTurn, Row: world[0]}
// 		bottomRequest := stubs.WorkerHaloRequest{LastCompletedTurn: lastCompletedTurn, Row: world[len(world)-1]}
// 		mu.Unlock()

// 		topResponse := new(stubs.WorkerHaloResponse)
// 		bottomResponse := new(stubs.WorkerHaloResponse)
// 		// These will deadlock, because the topClient and bottomClient are also at the same stage, calling their own top and bottom Clients, and noone will execute because it's
// 		// synchronous. I also haven't provided any validation for checking that the correct row is being sent for the correct turn
// 		topClient.Call("GameOfLife.SendTopHalo", topRequest, topResponse)
// 		bottomClient.Call("GameOfLife.SendBottomHalo", bottomRequest, bottomResponse)
// 		topRow := globalTopHalo
// 		bottomRow := globalBottomHalo

// 		outputChannelList := make([]chan [][]uint8, threads)
// 		for i := range outputChannelList {
// 			outputChannelList[i] = make(chan [][]uint8)
// 		}

// 		for i := 0; i < threads; i++ {
// 			var upperBound int
// 			if i == threads-1 {
// 				upperBound = request.EndY
// 			} else {
// 				upperBound = request.StartY + (i+1)*workerHeight
// 			}
// 			go worker(request.StartY+i*workerHeight, upperBound, 0, request.FullWorldWidth, world, topRow, bottomRow, outputChannelList[i])
// 		}

// 		// Recombine result of parallel execution
// 		var newWorld [][]uint8
// 		for i := 0; i < threads; i++ {
// 			newWorld = append(newWorld, <-outputChannelList[i]...)
// 		}

// 		mu.Lock()
// 		globalWorld = newWorld
// 		turn++
// 		lastCompletedTurn = turn
// 		mu.Unlock()
// 	}
// }
