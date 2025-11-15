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
	// for threads := 1; threads <= 16; threads++ {
	// os.Stdout = nil // Disable all program output apart from benchmark results
	p := gol.Params{
		Turns:       benchLength,
		Threads:     1,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	workers := 3
	name := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, workers)
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
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
	// }
}
