package main

import (
	"fmt"
	"strconv"
	"strings"
)

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	dx := []int{-1, 0, 1, 1, 1, 0, -1, -1}
	dy := []int{-1, -1, -1, 0, 1, 1, 1, 0}

	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {
		changes := []cell{}
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				// Placeholder for the actual Game of Life logic: flips alive cells to dead and dead cells to alive.
				nb := 0
				for i := 0; i < 8; i++ {
					c := x + dx[i]
					l := y + dy[i]
					if c == -1 {
						c = p.imageWidth - 1
					}
					if l == -1 {
						l = p.imageHeight - 1
					}
					if c == p.imageWidth {
						c = 0
					}
					if l == p.imageHeight {
						l = 0
					}
					if world[l][c] != 0 {
						nb++
					}

				}

				if world[y][x] != 0 {
					if nb < 2 || nb > 3 {
						changes = append(changes, cell{x, y})
						fmt.Println("cell", x, y, "is dead now.")
					}
				} else {
					if nb == 3 {
						changes = append(changes, cell{x, y})
						fmt.Println("cell", x, y, "is alive now.")
					}
				}

				//world[y][x] = world[y][x] ^ 0xFF
			}
		}
		for _, change := range changes {
			if world[change.y][change.x] != 0 {
				world[change.y][change.x] = 0
			} else {
				world[change.y][change.x] = 255
			}
		}
	}

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			d.io.outputVal <- world[y][x]
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
