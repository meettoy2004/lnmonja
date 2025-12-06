package protocol

// This file contains streaming protocol placeholders
// In production, streaming methods would be auto-generated from .proto files

// StreamContext provides context for streaming operations
type StreamContext interface {
	Context() interface{}
	Send(interface{}) error
	Recv() (interface{}, error)
}
