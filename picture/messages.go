package picture

// kittyFrameMsg carries the result of building a Kitty APC payload + grid for
// a specific generation of the Model. Update() ignores frames whose seq does
// not match the Model's current seq (image/size/mode changed since dispatch).
type kittyFrameMsg struct {
	seq  uint64
	apc  string
	grid string
}
