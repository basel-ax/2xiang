package domain

// Image represents an image generation request and its status
type Image struct {
	ID     int
	Prompt string
	UUID   string
	Status string
	Base64 string
}
