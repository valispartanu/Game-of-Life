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
func worker(p golParams, world [][]byte, turnDone chan bool, isdone chan bool, line1 int, line2 int, masterWG *sync.WaitGroup) {

	dx := []int{-1, 0, 1, 1, 1, 0, -1, -1}
	dy := []int{-1, -1, -1, 0, 1, 1, 1, 0}

	fmt.Println(line1, line2)

	for turns := 0; turns < p.turns; turns++ {
		//swg.Add(1)
		//fmt.Println(next)
		changes := []cell{}
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
						fmt.Println("cell", x, y, "is dead now.")
					}
				} else {
					if nb == 3 {
						changes = append(changes, cell{x, y})
						fmt.Println("cell", x, y, "is alive now.")
					}
				}
			}
		}

		isdone <- true
		<-turnDone
		//fmt.Println(line1, line2, myGroup, next, prev)
		//myGroup.Done()

		//next.Wait()
		//prev.Wait()

		for _, change := range changes {
			if world[change.y][change.x] != 0 {
				world[change.y][change.x] = 0
			} else {
				world[change.y][change.x] = 255
			}
		}
	}
	masterWG.Done()
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, k <-chan rune) {
	// Create the 2D slice to store the world.
	world := allocateSlice(p)

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	//var aliveNo = 0

	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}

	//	workerGroup := make([]sync.WaitGroup, p.threads)

	var wg sync.WaitGroup
	wg.Add(p.threads)
	//workerGroup[0].Add(1)
	//go worker(p, world, &workerGroup[p.threads-1], &workerGroup[0], &workerGroup[1], 0, (p.imageHeight / p.threads), &wg)

	turnDone := make([]chan bool, p.threads)
	for i := 0; i < p.threads; i++ {
		turnDone[i] = make(chan bool, 10)
	}
	isDone := make(chan bool, 16)

	for i := 0; i < p.threads; i++ {

		line1 := (p.imageHeight / p.threads) * i
		line2 := (p.imageHeight / p.threads) * (i + 1)
		//workerGroup[i].Add(1)
		go worker(p, world, turnDone[i], isDone, line1, line2, &wg)

	}

	for turn := 0; turn < p.turns; turn++ {

		for i := 0; i < p.threads; i++ {
			<-isDone
		}
		for i := 0; i < p.threads; i++ {
			turnDone[i] <- true
			fmt.Println("Notified", i, "-th  thread")
		}
		fmt.Println("all threads have finished za turn", turn)
	}

	/*	if p.threads > 1 {
		workerGroup[p.threads-1].Add(1)
		go worker(p, world, &workerGroup[p.threads-2], &workerGroup[p.threads-1], &workerGroup[0], (p.imageHeight/p.threads)*(p.threads-1), p.imageHeight, &wg)
	} */

	wg.Wait()
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
	fmt.Println(finalAlive)

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
