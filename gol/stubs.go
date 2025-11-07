package gol

import "uk.ac.bris.cs/gameoflife/util"

var loop = "GameOfLife.Loop"
var tickerService = "GameOfLife.TickerService"

type WorkerResponse struct {
	// Probably need to pass in turns, worker number, board, h and w etc.
	World      [][]uint8
	AliveCells []util.Cell
}

type WorkerRequest struct {
	// Probably need to pass in turns, worker number, board, etc.
	Turns  int
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
	World  [][]uint8
}

type TickerRequest struct {
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
}

type TickerResponse struct {
	CompletedTurns  int
	AliveCellsCount int
}
