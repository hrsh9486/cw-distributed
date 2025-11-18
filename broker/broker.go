package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// Functions to be called by the broker to access worker methods
var loop = "GameOfLife.Loop"

var mu sync.Mutex
var cond = sync.NewCond(&mu)

type Broker struct {
}

var globalWorkers []string
var globalWorld [][]uint8
var globalCompletedTurns int

var globalAliveCells []util.Cell
var globalThreads int
var quitting bool
var pausing bool
var killChan chan bool
var clients []*rpc.Client

func (broker Broker) RegisterWorker(request stubs.WorkerConnectionRequest, response *stubs.WorkerConnectionResponse) (err error) {
	mu.Lock()
	globalWorkers = append(globalWorkers, request.Address)
	mu.Unlock()
	return

}

func (broker Broker) ScheduleWork(request *stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	// Split up world, call worker methods, recollect

	fullWorldHeight := request.EndY - request.StartY
	fullWorldWidth := request.EndX - request.StartX
	sectionHeight := fullWorldHeight / len(globalWorkers)
	mu.Lock()
	globalCompletedTurns = 0
	globalWorld = stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth)
	globalAliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, globalWorld)
	quitting = false
	mu.Unlock()

	var wg sync.WaitGroup

	clients = make([]*rpc.Client, len(globalWorkers))
	for i := range globalWorkers {
		client, err := rpc.Dial("tcp", globalWorkers[i])
		if err != nil {
			fmt.Println(err)
		}
		defer client.Close()
		mu.Lock()
		clients[i] = client
		mu.Unlock()
	}
	turn := 0
	for turn < request.Turns && !quitting {
		newWorld := make([][]uint8, fullWorldHeight)
		responses := make([]*stubs.BrokerResponse, len(globalWorkers))
		for i := range globalWorkers {
			var upperBound int
			if i == len(globalWorkers)-1 {
				upperBound = len(globalWorld)
			} else {
				upperBound = (i + 1) * sectionHeight
			}

			mu.Lock()
			for pausing {
				cond.Wait()
			}
			mu.Unlock()

			wg.Add(1)
			go func(i, upperBound int) {
				defer wg.Done()
				req := stubs.BrokerRequest{
					StartY:          i * sectionHeight,
					EndY:            upperBound,
					StartX:          request.StartX,
					EndX:            request.EndX,
					FullWorldHeight: fullWorldHeight,
					FullWorldWidth:  fullWorldHeight,
					Threads:         globalThreads,
					BitMap:          stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth),
					Pausing:         pausing,
				}

				responses[i] = new(stubs.BrokerResponse)
				clients[i].Call(loop, &req, &responses[i])

			}(i, upperBound)

		}
		wg.Wait()
		if quitting {
			mu.Lock()
			response.CompletedTurns = globalCompletedTurns
			response.BitMap = stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth)
			response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
			mu.Unlock()
			return
		}

		mu.Lock()
		// Need to fix this section to ensure that they are appending it correctly
		for i := range globalWorkers {
			var upperBound int
			if i == len(globalWorkers)-1 {
				upperBound = len(globalWorld)
			} else {
				upperBound = (i + 1) * sectionHeight
			}
			res := responses[i]
			resWorld := stubs.Decode(res.BitMap, upperBound-(i*sectionHeight), fullWorldWidth)
			for row := i * sectionHeight; row < upperBound; row++ {
				newWorld[row] = resWorld[row-(i*sectionHeight)]
			}
		}

		globalWorld = newWorld
		globalCompletedTurns += 1
		globalAliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, newWorld)
		mu.Unlock()
		turn++
	}

	mu.Lock()
	if request.Turns == 0 {
		response.CompletedTurns = 0
		response.BitMap = request.BitMap
		response.AliveCells = getAliveCells(fullWorldHeight, fullWorldWidth, stubs.Decode(request.BitMap, fullWorldHeight, fullWorldWidth))
	} else {
		response.CompletedTurns = globalCompletedTurns
		response.BitMap = stubs.Encode(globalWorld, fullWorldHeight, fullWorldWidth)
		response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	}
	mu.Unlock()
	return
}

func (broker Broker) TickerService(request stubs.TickerRequest, response *stubs.TickerResponse) (err error) {
	mu.Lock()
	response.AliveCellsCount = len(getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld))
	response.CompletedTurns = globalCompletedTurns
	mu.Unlock()
	return
}

func (broker Broker) Quitter(request stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	mu.Lock()
	cond.Broadcast()
	pausing = false
	quitting = true
	response.BitMap = stubs.Encode(globalWorld, request.EndY-request.StartY, request.EndX-request.StartX)
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	response.CompletedTurns = globalCompletedTurns
	mu.Unlock()
	return
}

func (broker Broker) Killer(request stubs.ClientRequest, response *stubs.ClientResponse) (err error) {
	mu.Lock()
	pausing = false
	cond.Broadcast()

	// Gracefully shut down the broker and workers
	response.BitMap = stubs.Encode(globalWorld, request.EndY-request.StartY, request.EndX-request.StartX)
	response.AliveCells = getAliveCells(request.EndY-request.StartY, request.EndX-request.StartX, globalWorld)
	response.CompletedTurns = globalCompletedTurns

	quitting = true
	for i := range clients {
		req := stubs.WorkerConnectionRequest{}
		res := stubs.WorkerConnectionResponse{}
		clients[i].Call("GameOfLife.Killer", req, res)
	}

	killChan <- true
	mu.Unlock()
	return
}

func (broker Broker) Saver(request stubs.SaverRequest, response *stubs.SaverResponse) (err error) {
	mu.Lock()
	response.CompletedTurns = globalCompletedTurns
	response.BitMap = stubs.Encode(globalWorld, len(globalWorld), len(globalWorld[0]))
	mu.Unlock()
	return
}

func (broker Broker) Pauser(request stubs.PauserRequest, response *stubs.PauserResponse) (err error) {
	mu.Lock()
	pausing = !pausing
	response.CompletedTurns = globalCompletedTurns

	if !pausing {
		cond.Broadcast()
	}
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
	pAddr := flag.String("port", "8030", "The port the broker is listening on")
	remote := flag.String("remote", "0", "Is it running on a local instance?")
	threads := flag.String("threads", "1", "Specify number of threads to run on each worker")

	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	globalThreads, _ = strconv.Atoi(*threads)
	killChan = make(chan bool)

	var listenIP string

	if *remote == "0" {
		listenIP = "0.0.0.0"
		imdsHost := "169.254.169.254"
		myPrivateIP := stubs.GetMyIP(imdsHost, true)
		myPublicIP := stubs.GetMyIP(imdsHost, false)
		fmt.Println("Broker running on EC2 instance, with private IP: " + myPrivateIP)
		fmt.Println("Public IP:" + myPublicIP)
		fmt.Println("")
	} else {
		listenIP = "127.0.0.1"

	}

	rpc.Register(&Broker{})

	listener, err := net.Listen("tcp", listenIP+":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Broker listening on port:", listenIP+":"+*pAddr)
	go func() {
		<-killChan
		time.Sleep(1 * time.Second)
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Broker shut down gracefully")
			return
		}
		go rpc.ServeConn(conn)

	}
}
