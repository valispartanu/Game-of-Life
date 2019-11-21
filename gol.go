package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

func allocateSlice(p golParams) [][]byte {
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}
	return world
}
func worker(p golParams, input chan cell, changes chan cell, thread int) {

	world := allocateSlice(p)
	for {
		c, ok := <-input
		if ok == false {
			break
		} else {
			fmt.Println(c.y, " ", c.x)
			world[c.y][c.x] = 255
		}
	}
	line1 := (p.imageHeight / p.threads) * thread
	//fmt.Println("line1", line1)
	line2 := (p.imageHeight / p.threads) * (thread + 1)
	//fmt.Println("line2", line2)

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
					fmt.Println("cell", x, y, "is dead now.")
				}
			} else {
				if nb == 3 {
					changes <- cell{x, y}
					fmt.Println("cell", x, y, "is alive now.")
				}
			}
		}
	}
	//update(world, changes)
	//time.Sleep(2 * time.Second)
	close(changes)
}
func update(world [][]byte, output chan cell, wg *sync.WaitGroup) {

	for {
		c, ok := <-output
		if ok == false {
			break
		} else {
			if world[c.y][c.x] != 0 {
				world[c.y][c.x] = 0
			} else {
				world[c.y][c.x] = 255
			}
		}
	}
	wg.Done()

}

func sendData(output chan<- cell, world [][]byte, line1 int, line2 int, p golParams) {

	for i := line1; i < line2; i++ {
		for j := 0; j < p.imageWidth; j++ {
			if world[i][j] != 0 {
				c := cell{j, i}
				output <- c
			}
		}
	}
	if line1 == 0 {
		for j := 0; j < p.imageWidth; j++ {
			if world[p.imageHeight-1][j] != 0 {
				output <- cell{j, p.imageHeight - 1}
			}
		}
	} else {
		for j := 0; j < p.imageWidth; j++ {
			if world[line1-1][j] != 0 {
				output <- cell{j, line1 - 1}
			}
		}
	}

	if line2 == p.imageHeight {
		for j := 0; j < p.imageWidth; j++ {
			if world[0][j] != 0 {
				output <- cell{j, 0}
			}
		}
	} else {

		for j := 0; j < p.imageWidth; j++ {
			if world[line2][j] != 0 {
				output <- cell{j, line2}
			}
		}
	}
	fmt.Println("channel closed")

	close(output)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := allocateSlice(p)

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	for turns := 0; turns < p.turns; turns++ {
		// The io goroutine sends the requested image byte by byte, in rows.
		//changes := make(chan cell)

		chans := make([]chan cell, p.threads)
		for i := 0; i < p.threads; i++ {
			chans[i] = make(chan cell, 100)
		}

		for i := 0; i < p.threads; i++ {

			line1 := (p.imageHeight / p.threads) * i
			line2 := (p.imageHeight / p.threads) * (i + 1)
			input := make(chan cell, 100)
			sendData(input, world, line1, line2, p)
			fmt.Println("starting worker")
			go worker(p, input, chans[i], i)
		}

		var wg sync.WaitGroup
		for i := 0; i < p.threads; i++ {
			wg.Add(1)
			go update(world, chans[i], &wg)
		}
		wg.Wait()
		//time.Sleep(7*time.Second)
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

	d.io.command <- ioOutput
	d.io.filename <- "filename"
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
