package stubs

import "uk.ac.bris.cs/gameoflife/util"

// Functions to be called by workers to register with a broker
var registerWorker = "broker.RegisterWorker"

// Request and response for workers to register with client
// -------------------------------------------------------------------------------
type WorkerConnectionRequest struct {
	Address string // worker's address, e.g. "127.0.0.1:8050"
}

type WorkerConnectionResponse struct {
}

//-------------------------------------------------------------------------------

// Request and response for broker to call worker methods
// -------------------------------------------------------------------------------
type BrokerResponse struct {
	// Probably need to pass in turns, worker number, board, h and w etc.
	World          [][]uint8
	AliveCells     []util.Cell
	CompletedTurns int
}

type BrokerRequest struct {
	// Probably need to pass in turns, worker number, board, etc.
	Turns  int
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
	World  [][]uint8
}

//-------------------------------------------------------------------------------

// Request and response for client to access broker
// -------------------------------------------------------------------------------
type ClientRequest struct {
	Turns  int
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
	World  [][]uint8
}

type ClientResponse struct {
	World          [][]uint8
	AliveCells     []util.Cell
	CompletedTurns int
}

//-------------------------------------------------------------------------------

// Request and response for various miscellaneous such as getting alive cells for ticker
// -------------------------------------------------------------------------------
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

type KillerRequest struct{}
type KillerResponse struct{}

//-------------------------------------------------------------------------------
