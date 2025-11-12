package stubs

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

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

type BrokerRequest struct {
	// Probably need to pass in turns, worker number, board, etc.
	Turns  int
	StartY int
	EndY   int
	StartX int
	EndX   int
	H      int
	BitMap []byte
}

type BrokerResponse struct {
	// Probably need to pass in turns, worker number, board, h and w etc.
	BitMap         []byte
	CompletedTurns int
	StartY         int
	EndY           int
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
	BitMap []byte
}

type ClientResponse struct {
	BitMap         []byte
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
	BitMap         []byte
}

type KillerRequest struct{}
type KillerResponse struct{}

//-------------------------------------------------------------------------------

func Encode(game [][]uint8, h, w int) []byte {
	numBytes := (w*h + 7) / 8
	bitMap := make([]byte, numBytes)
	count := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			count += 1
			if game[y][x] == 255 {
				bitIndex := y*x + x
				byteIndex := bitIndex / 8
				bitPosition := uint(bitIndex % 8)
				bitMap[byteIndex] |= (1 << uint(bitPosition))
			}
		}
	}

	return bitMap
}

func Decode(bitMap []byte, h, w int) [][]uint8 {
	game := make([][]uint8, h)
	for i := range game {
		game[i] = make([]uint8, w)
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bitIndex := y*x + x
			byteIndex := bitIndex / 8
			bitPosition := uint(bitIndex % 8)
			if (bitMap[byteIndex] & (1 << bitPosition)) != 0 {
				game[y][x] = 255
			}
		}
	}
	return game
}

func GetMyPrivateIP(metadataHost string) string {
	client := http.Client{Timeout: 2 * time.Second}

	// Get IMDSv2 session token
	tokenReq, _ := http.NewRequest("PUT", fmt.Sprintf("http://%s/latest/api/token", metadataHost), nil)
	tokenReq.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", "60")
	tokenResp, _ := client.Do(tokenReq)
	defer tokenResp.Body.Close()
	token, _ := io.ReadAll(tokenResp.Body)

	// Use token to fetch private IP
	ipReq, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/latest/meta-data/local-ipv4", metadataHost), nil)
	ipReq.Header.Add("X-aws-ec2-metadata-token", string(token))
	ipResp, _ := client.Do(ipReq)
	defer ipResp.Body.Close()
	ipBytes, _ := io.ReadAll(ipResp.Body)

	return strings.TrimSpace(string(ipBytes))
}
