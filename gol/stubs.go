package gol

import "uk.ac.bris.cs/gameoflife/util"

var loop = "GameOfLife.Loop"

type Response struct {
	// Probably need to pass in turns, worker number, board, h and w etc.
	World      [][]uint8
	AliveCells []util.Cell
}

type Request struct {
	// Probably need to pass in turns, worker number, board, etc.
	Turns  int
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
	World  [][]uint8
}
