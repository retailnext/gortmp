package rtmpclient

type RTMPEvent struct {
	Data interface{}
}

type StatusEvent struct {
	Status ConnectionStatus
}

type ClosedEvent struct {
}

type VideoEvent struct {
	Message *Message
}

type AudioEvent struct {
	Message *Message
}

type CommandEvent struct {
	Command *Command
}

type MetadataEvent struct {
	AMFVersion AMFVersion
	Message    *Message
}

type UnknownDataEvent struct {
	Message *Message
}

type StreamCreatedEvent struct {
	Stream ClientStream
}

type StreamBegin struct {
	StreamID uint32
}

type StreamEOF struct {
	StreamID uint32
}

type StreamDry struct {
	StreamID uint32
}

type StreamIsRecorded struct {
	StreamID uint32
}

type AMFVersion int

const (
	AMF0 AMFVersion = 0
	AMF3 AMFVersion = 3
)
