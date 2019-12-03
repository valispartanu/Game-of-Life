package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func allocateSlice(p golParams) [][]byte {
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}
	return world
}

func worker(p golParams, input chan cell, thread int, above chan cell, below chan cell, final chan cell, turnDone chan bool, isdone chan bool, alive chan int, quit chan bool, state chan bool) {

	running := true
	//allocate empty world to worker
	world := allocateSlice(p)
	var ch = 0

	//receiving the initial configuration of his part
	c := <-input
	for c.x != -73 {
		world[c.y][c.x] = 255
		c = <-input
	}

	//defining the halos
	line1 := (p.imageHeight / p.threads) * thread
	line2 := (p.imageHeight / p.threads) * (thread + 1)

	//check for 6,10,12 threads
	if thread == p.threads-1 {
		line2 = p.imageHeight
	}

	dx := []int{-1, 0, 1, 1, 1, 0, -1, -1}
	dy := []int{-1, -1, -1, 0, 1, 1, 1, 0}

	for turn := 0; turn < p.turns; turn++ {

		if running == true {

			//make future halos empty
			if line1 == 0 {
				for j := 0; j < p.imageWidth; j++ {
					world[p.imageHeight-1][j] = 0
				}
			} else {
				for j := 0; j < p.imageWidth; j++ {
					world[line1-1][j] = 0
				}
			}

			if line2 == p.imageHeight {
				for j := 0; j < p.imageWidth; j++ {
					world[0][j] = 0
				}
			} else {
				for j := 0; j < p.imageWidth; j++ {
					world[line2][j] = 0
				}
			}

			done := make(chan bool)

			//receives configuration for halos
			go func() {
				c = <-input
				for c.x != -73 {
					world[c.y][c.x] = 255
					c = <-input
				}

				c = <-input
				for c.x != -73 {
					world[c.y][c.x] = 255
					c = <-input
				}
				done <- true
			}()

			//send halo to the above and below workers
			sendData(above, world, line1+1, line1, p)
			sendData(below, world, line2, line2-1, p)

			<-done

			var changes []cell

			//Logic of the game
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
							changes = append(changes, cell{x, y})
						}
					} else {
						if nb == 3 {
							changes = append(changes, cell{x, y})
						}
					}
				}
			}
			ch = 0

			//update
			for _, change := range changes {
				if world[change.y][change.x] != 0 {
					world[change.y][change.x] = 0
					ch--
				} else {
					world[change.y][change.x] = 255
					ch++
				}
			}
			alive <- ch
			//notifies the distributor that the work is done for this turn on his part
			isdone <- true
			//waits signal from distributor that the rest of workers finished their work
			<-turnDone
		}
		select {
		case <-quit:
			turn = p.turns + 1
			running = false
		case <-state:
			sendData(final, world, line1+1, line2-1, p)
		default:
		}

	}
	//send updated part to distributor
	sendData(final, world, line1+1, line2-1, p)
}
func update(world [][]byte, output chan cell) {
	c := <-output
	for c.x != -73 {
		world[c.y][c.x] = 255
		c = <-output
	}
}

func sendData(output chan cell, world [][]byte, line1 int, line2 int, p golParams) {

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
	output <- cell{-73, -73}
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

	ticker := time.NewTicker(2 * time.Second)

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

	isDone := make(chan bool, 16)
	final := make(chan cell, 10)

	//creating a channel to send the number of alive cells
	al := make(chan int, p.threads)

	//slice of channel to send and receive for each worker
	chans := make([]chan cell, p.threads)
	for i := 0; i < p.threads; i++ {
		chans[i] = make(chan cell, 10)
	}

	//slice of bool channels for each worker to send "true" if it finished the turn
	turnDone := make([]chan bool, p.threads)
	for i := 0; i < p.threads; i++ {
		turnDone[i] = make(chan bool, 10)
	}

	//slice of bool channels for each worker to notify quitting when quit command pressed
	quit := make([]chan bool, p.threads)
	for i := 0; i < p.threads; i++ {
		quit[i] = make(chan bool, 2)
	}

	state := make([]chan bool, p.threads)
	for i := 0; i < p.threads; i++ {
		state[i] = make(chan bool, 2)
	}

	//first worker
	line1 := (p.imageHeight / p.threads) * 0
	line2 := (p.imageHeight / p.threads) * (0 + 1)
	go worker(p, chans[0], 0, chans[p.threads-1], chans[1], final, turnDone[0], isDone, al, quit[0], state[0])
	sendData(chans[0], world, line1, line2, p)

	//middle workers
	for i := 1; i < p.threads-1; i++ {
		line1 := (p.imageHeight / p.threads) * i
		line2 := (p.imageHeight / p.threads) * (i + 1)
		//input := make(chan cell, 10)
		go worker(p, chans[i], i, chans[i-1], chans[i+1], final, turnDone[i], isDone, al, quit[i], state[i])
		sendData(chans[i], world, line1, line2, p)
	}

	//last worker
	if p.threads > 1 {
		line1 := (p.imageHeight / p.threads) * (p.threads - 1)
		line2 := p.imageHeight
		go worker(p, chans[p.threads-1], p.threads-1, chans[p.threads-2], chans[0], final, turnDone[p.threads-1], isDone, al, quit[p.threads-1], state[p.threads-1])
		sendData(chans[p.threads-1], world, line1, line2, p)
	}

	spressed := false

	for turn := 0; turn < p.turns; turn++ {

		if paused == true && running == true {
			turn--
		}

		for i := 0; i < p.threads; i++ {
			//each worker notifies the distributor that it finished his work on the current turn
			<-isDone
		}
		select {
		case ch := <-k:
			if ch == 'q' && running == true {
				running = false
				fmt.Println("Quitting")
				for i := 0; i < p.threads; i++ {
					quit[i] <- true
				}
				turn = p.turns
				for i := 0; i < p.threads; i++ {
					turnDone[i] <- true
				}
			}

			if ch == 's' {
				spressed = true
				for i := 0; i < p.threads; i++ {
					state[i] <- true
				}
			}

			if ch == 'p' {
				if paused == true {
					fmt.Println("Continuing")
					paused = false
				} else {
					fmt.Println("Paused")
					paused = true
				}
			}
		default:
		}

		if running == true {

			for i := 0; i < p.threads; i++ {
				turnDone[i] <- true
				aliveNo += <-al
			}

			if spressed == true {
				spressed = false
				world = allocateSlice(p)
				for i := 0; i < p.threads; i++ {
					update(world, final)
				}
				printPGM(world, d, p)
			}
		}
	}

	//construct the final configuration of the matrix
	world = allocateSlice(p)
	for i := 0; i < p.threads; i++ {
		update(world, final)
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
