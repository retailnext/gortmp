package gortmp

import "io"

type RTMPEvent struct {
	Data interface{}
}

type StatusEvent struct {
	Status ConnectionStatus
}

type ClosedEvent struct {
}

type VideoEvent struct {
	Timestamp uint32
	Data      io.Reader
}

type AudioEvent struct {
	Timestamp uint32
	Data      io.Reader
}

type CommandEvent struct {
	Command *Command
}

type StreamCreatedEvent struct {
	Stream ClientStream
}

type UnknownDataEvent struct {
	Message *Message
}
