package main

import (
	"fmt"
	"os"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

const benchLength = 1000

func BenchmarkGol(b *testing.B) {
	os.Stdout = nil
	// os.Stdout = nil // Disable all program output apart from benchmark results
	p := gol.Params{
		Turns:       benchLength,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	name := fmt.Sprintf("DistributedMultithread-%dx%dx%d-1", p.ImageWidth, p.ImageHeight, p.Turns)
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			os.Stdout = nil
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
			for range events {
				for event := range events {
					switch event.(type) {
					case gol.FinalTurnComplete:
					}
				}
			}
		}
	})
}
