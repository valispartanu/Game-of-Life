package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

func allocateSlice(p golParams) [][]byte {
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}
	return world
}
func worker(p golParams, input chan cell, changes chan cell, thread int) {
	//allocate empty world to the worker
	world := allocateSlice(p)

	//receive the initial configuration of the worker's part of world
	for {
		c, ok := <-input
		if ok == false {
			break
		} else {
			world[c.y][c.x] = 255
		}
	}

	//define bounds for each worker
	line1 := (p.imageHeight / p.threads) * thread
	line2 := (p.imageHeight / p.threads) * (thread + 1)

	//check for 6,10,12 threads
	if thread == p.threads-1 {
		line2 = p.imageHeight
	}

	//game logic
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
				}
			} else {
				if nb == 3 {
					changes <- cell{x, y}
				}
			}
		}
	}
	close(changes)
}

//flips dead to alive and alive to dead depending on what coordinates it gets from changes
func update(world [][]byte, output chan cell, wg *sync.WaitGroup, alive chan int) {

	var ch = 0
	for {
		c, ok := <-output
		if ok == false {
			break
		} else {
			if world[c.y][c.x] != 0 {
				world[c.y][c.x] = 0
				ch--
			} else {
				world[c.y][c.x] = 255
				ch++
			}
		}
	}
	alive <- ch
	//decreasing the WaitGroup counter by 1 for each update
	wg.Done()
}

//send coordinates of alive cells to the worker (output->input)
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
	close(output)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, k <-chan rune) {
	// Create the 2D slice to store the world.
	world := allocateSlice(p)

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	var aliveNo = 0

	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
				aliveNo++
			}
		}
	}
	var running = true
	var paused = false

	//creating a 2 second ticker
	ticker := time.NewTicker(2 * time.Second)

	//printing the number of alive cells every 2 seconds (when receiving from the ticker channel)
	go func() {
		for {
			select {
			case <-ticker.C:
				if running == true && paused == false {
					fmt.Println(aliveNo)
				}
			}
		}
	}()

	for turns := 0; turns < p.turns && running == true; turns++ {

		//selecting between commands in make gol using the key channel, k
		select {
		case ch := <-k:
			if ch == 'q' {
				running = false
			}
			if ch == 's' {
				printPGM(world, d, p)
			}
			if ch == 'p' {
				if paused == true {
					paused = false
				} else {
					paused = true
				}
			}
		default:
		}

		if paused == true {
			turns--
		} else {
			chans := make([]chan cell, p.threads)
			for i := 0; i < p.threads; i++ {
				chans[i] = make(chan cell, 10)
			}

			for i := 0; i < p.threads; i++ {
				line1 := (p.imageHeight / p.threads) * i
				line2 := (p.imageHeight / p.threads) * (i + 1)

				//check for 6,10,12 threads
				if i == p.threads-1 {
					line2 = p.imageHeight
				}
				input := make(chan cell, 10)
				go worker(p, input, chans[i], i)
				sendData(input, world, line1, line2, p)
			}
			//creating a channel to send the number of alive cells
			al := make(chan int, p.threads)
			var wg sync.WaitGroup
			for i := 0; i < p.threads; i++ {
				//adding 1 to the WaitGroup counter for each worker update in this turn
				wg.Add(1)
				go update(world, chans[i], &wg, al)
			}
			//verifying that all updates are done for this turn
			wg.Wait()

			//updating the number of alive cells each turn
			for i := 0; i < p.threads; i++ {
				aliveNo += <-al
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

	printPGM(world, d, p)
	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}

func printPGM(world [][]byte, d distributorChans, p golParams) {
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			d.io.outputVal <- world[y][x]
		}
	}
}
