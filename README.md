# Game-of-Life

This Go program involves the development of a concurrent version of John Conway’s Game of Life. One interacts with the game by creating an initial configuration (matrix of dead/alive cells) and observing how it evolves based on some rules. 

The first step in achieving a parallel and therefore a faster version of the game was to split the matrix between several worker functions. These were responsible for the game logic and for updates after each turn. 

A halo exchange scheme was also necessary in order to make the workers interact with each other when making an update. 

A user interaction feature was added, so that the user can pause, quit or request the current state of the board at any time of the execution.  
