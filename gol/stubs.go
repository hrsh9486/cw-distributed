package gol

import "uk.ac.bris.cs/gameoflife/util"

var loop = "GameOfLife.Loop"
var tickerService = "GameOfLife.TickerService"
var saver = "GameOfLife.Saver"
var quitter = "GameOfLife.Quitter"
var pauser = "GameOfLife.Pauser"
var killer = "GameOfLife.Killer"

type WorkerResponse struct {
	// Probably need to pass in turns, worker number, board, h and w etc.
	World          [][]uint8
	AliveCells     []util.Cell
	CompletedTurns int
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

type PauserRequest struct {
}

type PauserResponse struct {
	CompletedTurns int
}

type SaverRequest struct {
}

type SaverResponse struct {
	CompletedTurns int
	World          [][]uint8
}
