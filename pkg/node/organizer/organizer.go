package organizer

// TODO:
// every time there's a new file, this package should have a goroutine that:
// - splits the file into pieces
// - create a meta file
// - talks to the scheduler to register the new meta file and peer
