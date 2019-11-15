package main

import (
	"fmt"
	"strconv"
	"strings"
)

func allocateSlice(p golParams) [][]byte {
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}
	return world
}
func worker(p golParams, input chan cell, changes chan cell, thread int, nca int) {

	fmt.Println("Starting worker")
	world := allocateSlice(p)
	for i := 1; i <= nca; i++ {
		c := <-input
		world[c.y][c.x] = 255
	}
	fmt.Println("Starting logic")
	line1 := (p.imageHeight / p.threads) * thread
	line2 := (p.imageHeight / p.threads) * (thread + 1)

	//nc := 0

	dx := []int{-1, 0, 1, 1, 1, 0, -1, -1}
	dy := []int{-1, -1, -1, 0, 1, 1, 1, 0}
	for y := line1; y < line2; y++ {
		for x := 0; x < p.imageWidth; x++ {
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
					changes <- cell{x, y}
					//nc++
					fmt.Println("cell", x, y, "is dead now.")
				}
			} else {
				if nb == 3 {
					changes <- cell{x, y}
					//nc++
					fmt.Println("cell", x, y, "is alive now.")
				}
			}
		}
	}

}
func update(world [][]byte, output chan cell) {

	for {
		c := <-output
		if world[c.y][c.x] != 0 {
			world[c.y][c.x] = 0
		} else {
			world[c.y][c.x] = 255
		}
	}

}

func sendData(output chan cell, world [][]byte, line1 int, line2 int, p golParams) int {

	fmt.Println("Starting sending", line1, line2)
	nca := 0
	for i := line1; i < line2; i++ {
		fmt.Println("Starting sending", i)
		for j := 0; j < p.imageWidth; j++ {
			if world[i][j] != 0 {
				nca++
				output <- cell{j, i}
			}
		}
	}

	fmt.Println("Starting sending")
	if line1 == 0 {
		for j := 0; j < p.imageWidth; j++ {
			if world[p.imageHeight-1][j] != 0 {
				nca++
				output <- cell{j, p.imageHeight - 1}
			}
		}
	} else {
		for j := 0; j < p.imageWidth; j++ {
			if world[line1-1][j] != 0 {
				nca++
				output <- cell{j, line1 - 1}
			}
		}
	}

	if line2 == p.imageHeight {
		for j := 0; j < p.imageWidth; j++ {
			if world[0][j] != 0 {
				nca++
				output <- cell{j, 0}
			}
		}
	} else {

		for j := 0; j < p.imageWidth; j++ {
			if world[line2][j] != 0 {
				nca++
				output <- cell{j, line2}
			}
		}
	}
	return nca

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := allocateSlice(p)
	/*:= make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}*/

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

	//TODO send cells to each worked using sendata function, split lines between them, collect changes int changes channel, update the world

	changes := make(chan cell)
	//nc := 0
	fmt.Println("Starting goroutinrs")
	for i := 0; i < p.threads; i++ {

		line1 := (p.imageHeight / p.threads) * i
		line2 := (p.imageHeight / p.threads) * (i + 1)
		input := make(chan cell)
		nca := sendData(input, world, line1, line2, p)
		go worker(p, input, changes, i, nca)
		//go worker(p, alive)
	}
	fmt.Println("Finished goroutinrs")
	go update(world, changes)

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

	d.io.command <- ioOutput
	d.io.filename <- "filename.pgm"
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
