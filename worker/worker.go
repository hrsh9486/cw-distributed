package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

// Need a mutex lock on world and turn
type GameOfLife struct {
}

var killChan chan bool

func (Game GameOfLife) Killer(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	killChan <- true
	return
}

// Calculate a certain number of game of life states
func (Game GameOfLife) Loop(request stubs.BrokerRequest, response *stubs.BrokerResponse) (err error) {
	// quitting = false
	// pausing = false
	// isKilled = false
	// turn := 0
	world := stubs.Decode(request.BitMap, request.FullWorldHeight, request.FullWorldWidth)

	threads := request.Threads
	workerHeight := (request.EndY - request.StartY) / threads

	if threads == 1 {
		world = calculateNextState(request.StartY, request.EndY, 0, request.FullWorldWidth, world)
	} else {
		outputChannelList := make([]chan [][]uint8, threads)
		for i := range outputChannelList {
			outputChannelList[i] = make(chan [][]uint8)

		}
		for i := 0; i < threads; i++ {
			var upperBound int
			if i == threads-1 {
				upperBound = request.EndY
			} else {
				upperBound = request.StartY + (i+1)*workerHeight
			}
			go worker(request.StartY+i*workerHeight, upperBound, 0, request.FullWorldWidth, world, outputChannelList[i])

		}
		// Recombine result of parallel execution in new slice
		var newWorld [][]uint8
		for i := 0; i < threads; i++ {
			newWorld = append(newWorld, <-outputChannelList[i]...)
		}

		world = newWorld
	}
	response.BitMap = stubs.Encode(world, request.EndY-request.StartY, request.EndX-request.StartX)
	return

}

func worker(startY, endY, startX, endX int, world [][]uint8, outputChan chan [][]uint8) {
	outputChan <- calculateNextState(startY, endY, startX, endX, world)
}

// Take a broker state and iteratively calculate the next state for a section of the board
func calculateNextState(startY, endY, startX, endX int, world [][]uint8) [][]uint8 {
	w := len(world[0])
	h := len(world)
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

func main() {
	pAddr := flag.String("port", "8031", "Port the worker listens on")
	brokerAddr := flag.String("broker", "127.0.0.1", "IP address of the broker")
	remote := flag.String("remote", "0", "Is it running on a local instance?")

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	killChan = make(chan bool)
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
	go func() {
		<-killChan
		time.Sleep(1 * time.Second)
		listener.Close()
	}()

	fmt.Println("Worker listening on", listenAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Worker exited gracefully")
			return
		}
		go rpc.ServeConn(conn)
	}

}
