package rtmpclient

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/zhangpeihao/goamf"
)

type ConnectionStatus uint

func (s ConnectionStatus) String() string {
	var connectionStatusNames = map[ConnectionStatus]string{
		ConnectionStatusClose:          "StatusClose",
		ConnectionStatusHandshakeOK:    "StatusHandshakeOK",
		ConnectionStatusConnect:        "StatusConnect",
		ConnectionStatusConnectOK:      "StatusConnectOK",
		ConnectionStatusCreateStream:   "StatusCreateStream",
		ConnectionStatusCreateStreamOK: "StatusCreateStreamOK",
	}

	if str, ok := connectionStatusNames[s]; ok {
		return str
	}

	return "Unknown"
}

const (
	ConnectionStatusClose          ConnectionStatus = 0
	ConnectionStatusHandshakeOK                     = 1
	ConnectionStatusConnect                         = 2
	ConnectionStatusConnectOK                       = 3
	ConnectionStatusCreateStream                    = 4
	ConnectionStatusCreateStreamOK                  = 5
)

type ClientConn interface {
	// Connect an appliction on FMS after handshake.
	Connect(extendedParameters ...interface{}) (err error)
	// Create a stream
	CreateStream() (err error)
	// Close a connection
	Close()
	// URL to connect
	URL() string
	// Connection status
	Status() (ConnectionStatus, error)
	// Send a message
	Send(message *Message) error
	// Calls a command or method on Flash Media Server
	// or on an application server running Flash Remoting.
	Call(name string, customParameters ...interface{}) (err error)
	// Get network connect instance
	Conn() Conn
	// Returns a channel of RTMPEvents
	Events() <-chan RTMPEvent
}

// High-level interface
//
// A RTMP connection(based on TCP) to RTMP server(FMS or crtmpserver).
// In one connection, we can create many chunk streams.
type clientConn struct {
	url          string
	rtmpURL      RtmpURL
	status       ConnectionStatus
	err          error
	conn         Conn
	transactions map[uint32]string
	streams      map[uint32]ClientStream
	events       chan RTMPEvent
}

// Connect to FMS server, and finish handshake process
func Dial(url string, maxChannelNumber int) (ClientConn, error) {
	rtmpURL, err := ParseURL(url)
	if err != nil {
		return nil, err
	}
	var c net.Conn
	switch rtmpURL.protocol {
	case "rtmp":
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", rtmpURL.host, rtmpURL.port))
	case "rtmps":
		c, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", rtmpURL.host, rtmpURL.port), &tls.Config{InsecureSkipVerify: true})
	default:
		err = errors.New(fmt.Sprintf("Unsupport protocol %s", rtmpURL.protocol))
	}
	if err != nil {
		return nil, err
	}

	ipConn, ok := c.(*net.TCPConn)
	if ok {
		ipConn.SetWriteBuffer(128 * 1024)
	}
	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	timeout := time.Duration(10 * time.Second)
	err = Handshake(c, br, bw, timeout)
	if err == nil {
		logger.ModulePrintln(LOG_LEVEL_DEBUG, "Handshake OK")

		conn := &clientConn{
			url:          url,
			rtmpURL:      rtmpURL,
			status:       ConnectionStatusHandshakeOK,
			transactions: make(map[uint32]string),
			streams:      make(map[uint32]ClientStream),
			events:       make(chan RTMPEvent, 10),
		}
		conn.sendEvent(&StatusEvent{Status: conn.status})
		conn.conn = NewConn(c, br, bw, conn, maxChannelNumber)
		return conn, nil
	}

	return nil, err
}

// Connect to FMS server, and finish handshake process
func NewOutbounConn(c net.Conn, url string, maxChannelNumber int) (ClientConn, error) {
	rtmpURL, err := ParseURL(url)
	if err != nil {
		return nil, err
	}
	if rtmpURL.protocol != "rtmp" {
		return nil, errors.New(fmt.Sprintf("Unsupport protocol %s", rtmpURL.protocol))
	}

	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	conn := &clientConn{
		url:          url,
		rtmpURL:      rtmpURL,
		status:       ConnectionStatusHandshakeOK,
		transactions: make(map[uint32]string),
		streams:      make(map[uint32]ClientStream),
		events:       make(chan RTMPEvent, 10),
	}
	conn.conn = NewConn(c, br, bw, conn, maxChannelNumber)
	return conn, nil
}

// Connect an appliction on FMS after handshake.
func (conn *clientConn) Connect(extendedParameters ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if conn.err == nil {
				conn.err = err
			}
		}
	}()
	// Create connect command
	buf := new(bytes.Buffer)
	// Command name
	_, err = amf.WriteString(buf, "connect")
	CheckError(err, "Connect() Write name: connect")
	transactionID := conn.conn.NewTransactionID()
	conn.transactions[transactionID] = "connect"
	_, err = amf.WriteDouble(buf, float64(transactionID))
	CheckError(err, "Connect() Write transaction ID")
	_, err = amf.WriteObjectMarker(buf)
	CheckError(err, "Connect() Write object marker")

	_, err = amf.WriteObjectName(buf, "app")
	CheckError(err, "Connect() Write app name")
	_, err = amf.WriteString(buf, conn.rtmpURL.App())
	CheckError(err, "Connect() Write app value")

	_, err = amf.WriteObjectName(buf, "flashVer")
	CheckError(err, "Connect() Write flashver name")
	_, err = amf.WriteString(buf, FLASH_PLAYER_VERSION_STRING)
	CheckError(err, "Connect() Write flashver value")

	_, err = amf.WriteObjectName(buf, "tcUrl")
	CheckError(err, "Connect() Write tcUrl name")
	_, err = amf.WriteString(buf, conn.url)
	CheckError(err, "Connect() Write tcUrl value")

	_, err = amf.WriteObjectName(buf, "fpad")
	CheckError(err, "Connect() Write fpad name")
	_, err = amf.WriteBoolean(buf, false)
	CheckError(err, "Connect() Write fpad value")

	_, err = amf.WriteObjectName(buf, "capabilities")
	CheckError(err, "Connect() Write capabilities name")
	_, err = amf.WriteDouble(buf, DEFAULT_CAPABILITIES)
	CheckError(err, "Connect() Write capabilities value")

	_, err = amf.WriteObjectName(buf, "audioCodecs")
	CheckError(err, "Connect() Write audioCodecs name")
	_, err = amf.WriteDouble(buf, DEFAULT_AUDIO_CODECS)
	CheckError(err, "Connect() Write audioCodecs value")

	_, err = amf.WriteObjectName(buf, "videoCodecs")
	CheckError(err, "Connect() Write videoCodecs name")
	_, err = amf.WriteDouble(buf, DEFAULT_VIDEO_CODECS)
	CheckError(err, "Connect() Write videoCodecs value")

	_, err = amf.WriteObjectName(buf, "videoFunction")
	CheckError(err, "Connect() Write videoFunction name")
	_, err = amf.WriteDouble(buf, float64(1))
	CheckError(err, "Connect() Write videoFunction value")

	_, err = amf.WriteObjectEndMarker(buf)
	CheckError(err, "Connect() Write ObjectEndMarker")

	// extended parameters
	for _, param := range extendedParameters {
		_, err = amf.WriteValue(buf, param)
		CheckError(err, "Connect() Write extended parameters")
	}
	connectMessage := &Message{
		ChunkStreamID: CS_ID_COMMAND,
		Type:          COMMAND_AMF0,
		Size:          uint32(buf.Len()),
		Buf:           buf,
	}
	connectMessage.Dump("connect")
	conn.status = ConnectionStatusConnect
	return conn.conn.Send(connectMessage)
}

// Close a connection
func (conn *clientConn) Close() {
	for _, stream := range conn.streams {
		stream.Close()
	}
	conn.status = ConnectionStatusClose
	go func() {
		time.Sleep(time.Second)
		conn.conn.Close()
	}()
}

// URL to connect
func (conn *clientConn) URL() string {
	return conn.url
}

// Connection status
func (conn *clientConn) Status() (ConnectionStatus, error) {
	return conn.status, conn.err
}

// Callback when recieved message. Audio & Video data
func (conn *clientConn) OnReceived(c Conn, message *Message) {
	stream, found := conn.streams[message.StreamID]
	sendEvent := false
	if found {
		if !stream.Received(message) {
			sendEvent = true
		}
	} else {
		sendEvent = true
	}

	if sendEvent {
		switch message.Type {
		case VIDEO_TYPE:
			conn.sendEvent(&VideoEvent{Timestamp: message.Timestamp, Data: message.Buf})
		case AUDIO_TYPE:
			conn.sendEvent(&AudioEvent{Timestamp: message.Timestamp, Data: message.Buf})
		default:
			// This shouldn't get called. All event types should be handled
			conn.sendEvent(&UnknownDataEvent{Message: message})
		}
	}
}

func (conn *clientConn) Events() <-chan RTMPEvent {
	return conn.events
}

func (conn *clientConn) sendEvent(e interface{}) {
	conn.events <- RTMPEvent{
		Data: e,
	}
}

// Callback when recieved message.
func (conn *clientConn) OnReceivedRtmpCommand(c Conn, command *Command) {
	command.Dump()
	switch command.Name {
	case "_result":
		transaction, found := conn.transactions[command.TransactionID]
		if found {
			switch transaction {
			case "connect":
				if command.Objects != nil && len(command.Objects) >= 2 {
					information, ok := command.Objects[1].(amf.Object)
					if ok {
						code, ok := information["code"]
						if ok && code == RESULT_CONNECT_OK {
							// Connect OK
							conn.conn.SetWindowAcknowledgementSize()
							conn.status = ConnectionStatusConnectOK
							conn.sendEvent(&StatusEvent{Status: conn.status})
							conn.status = ConnectionStatusCreateStream
							conn.CreateStream()
						}
					}
				}
			case "createStream":
				if command.Objects != nil && len(command.Objects) >= 2 {
					streamID, ok := command.Objects[1].(float64)
					if ok {
						newChunkStream, err := conn.conn.CreateMediaChunkStream()
						if err != nil {
							logger.ModulePrintf(LOG_LEVEL_WARNING,
								"clientConn::ReceivedRtmpCommand() CreateMediaChunkStream err:", err)
							return
						}
						stream := &clientStream{
							id:            uint32(streamID),
							conn:          conn,
							chunkStreamID: newChunkStream.ID,
						}
						conn.streams[stream.ID()] = stream
						conn.status = ConnectionStatusCreateStreamOK
						conn.sendEvent(&StatusEvent{Status: conn.status})
						conn.sendEvent(&StreamCreatedEvent{Stream: stream})
					}
				}
			}
			delete(conn.transactions, command.TransactionID)
		}
	case "_error":
		transaction, found := conn.transactions[command.TransactionID]
		if found {
			logger.ModulePrintf(LOG_LEVEL_TRACE,
				"Command(%d) %s error\n", command.TransactionID, transaction)
		} else {
			logger.ModulePrintf(LOG_LEVEL_TRACE,
				"Command(%d) not been found\n", command.TransactionID)
		}
	case "onBWCheck":
	}
	conn.sendEvent(&CommandEvent{Command: command})
}

// Connection closed
func (conn *clientConn) OnClosed(c Conn) {
	conn.status = ConnectionStatusClose
	conn.sendEvent(&StatusEvent{Status: conn.status})
	conn.sendEvent(&ClosedEvent{})
}

// Create a stream
func (conn *clientConn) CreateStream() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if conn.err == nil {
				conn.err = err
			}
		}
	}()
	// Create createStream command
	transactionID := conn.conn.NewTransactionID()
	cmd := &Command{
		IsFlex:        false,
		Name:          "createStream",
		TransactionID: transactionID,
		Objects:       make([]interface{}, 1),
	}
	cmd.Objects[0] = nil
	buf := new(bytes.Buffer)
	err = cmd.Write(buf)
	CheckError(err, "createStream() Create command")
	conn.transactions[transactionID] = "createStream"

	message := &Message{
		ChunkStreamID: CS_ID_COMMAND,
		Type:          COMMAND_AMF0,
		Size:          uint32(buf.Len()),
		Buf:           buf,
	}
	message.Dump("createStream")
	return conn.conn.Send(message)
}

// Send a message
func (conn *clientConn) Send(message *Message) error {
	return conn.conn.Send(message)
}

// Calls a command or method on Flash Media Server
// or on an application server running Flash Remoting.
func (conn *clientConn) Call(name string, customParameters ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if conn.err == nil {
				conn.err = err
			}
		}
	}()
	// Create command
	transactionID := conn.conn.NewTransactionID()
	cmd := &Command{
		IsFlex:        false,
		Name:          name,
		TransactionID: transactionID,
		Objects:       make([]interface{}, 1+len(customParameters)),
	}
	cmd.Objects[0] = nil
	for index, param := range customParameters {
		cmd.Objects[index+1] = param
	}
	buf := new(bytes.Buffer)
	err = cmd.Write(buf)
	CheckError(err, "Call() Create command")
	conn.transactions[transactionID] = name

	message := &Message{
		ChunkStreamID: CS_ID_COMMAND,
		Type:          COMMAND_AMF0,
		Size:          uint32(buf.Len()),
		Buf:           buf,
	}
	message.Dump(name)
	return conn.conn.Send(message)

}

// Get network connect instance
func (conn *clientConn) Conn() Conn {
	return conn.conn
}
