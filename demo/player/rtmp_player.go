package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	rtmp "github.com/retailnext/rtmpclient"
)

const (
	programName = "RtmpPlayer"
	version     = "0.0.1"
)

var (
	url        *string = flag.String("URL", "rtmp://192.168.20.111/vid3", "The rtmp url to connect.")
	streamName *string = flag.String("Stream", "camstream", "Stream name to play.")
)

var obConn rtmp.ClientConn
var createStreamChan chan rtmp.ClientStream
var videoDataSize int64
var audioDataSize int64
var status uint

var h264StartCode = []byte{0x00, 0x00, 0x00, 0x01}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s version[%s]\r\nUsage: %s [OPTIONS]\r\n", programName, version, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	createStreamChan = make(chan rtmp.ClientStream)
	fmt.Println("to dial")

	var err error

	conn, err := rtmp.Dial(*url, 100)
	if err != nil {
		fmt.Println("Dial error", err)
		os.Exit(-1)
	}

	f, err := os.Create("/tmp/video_dump")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	defer conn.Close()
	fmt.Printf("conn: %+v\n", conn)
	fmt.Printf("conn.URL(): %s\n", conn.URL())
	fmt.Println("to connect")
	err = conn.Connect()
	if err != nil {
		fmt.Printf("Connect error: %s", err.Error())
		os.Exit(-1)
	}

	for done := false; !done; {
		select {
		case msg, ok := <-conn.Events():
			if !ok {
				done = true
			}
			switch ev := msg.Data.(type) {
			case *rtmp.StatusEvent:
				fmt.Println("evt status:", ev.Status)
			case *rtmp.ClosedEvent:
				fmt.Println("evt closed")
			case *rtmp.VideoEvent:
				n, _ := io.Copy(ioutil.Discard, ev.Data)
				fmt.Println("evt video:", ev.Timestamp, n)
			case *rtmp.AudioEvent:
				fmt.Println("evt audio")
			case *rtmp.CommandEvent:
				fmt.Println("evt command")
			case *rtmp.StreamCreatedEvent:
				fmt.Println("evt stream created")
				err = ev.Stream.Play(*streamName, nil, nil, nil)
				if err != nil {
					fmt.Printf("Play error: %s", err.Error())
					os.Exit(-1)
				}
			}
		case <-time.After(1 * time.Second):
			fmt.Printf("Audio size: %d bytes; Vedio size: %d bytes\n", audioDataSize, videoDataSize)
			return
		}
	}
}
