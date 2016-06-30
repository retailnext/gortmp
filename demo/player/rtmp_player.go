package main

import (
	"flag"
	"fmt"
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

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s version[%s]\r\nUsage: %s [OPTIONS]\r\n", programName, version, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

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

	streamIDs := make([]uint32, 0)

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
				fmt.Println("evt video:", ev.Message.Timestamp, ev.Message.Buf.Len())
			case *rtmp.AudioEvent:
				fmt.Println("evt audio")
			case *rtmp.CommandEvent:
				fmt.Println("evt command")
			case *rtmp.StreamBegin:
				fmt.Println("case *rtmp.StreamBegin")
			case *rtmp.StreamEOF:
				fmt.Println("case *rtmp.StreamEOF", ev.StreamID, streamIDs)
				return
			case *rtmp.StreamDry:
				fmt.Println("case *rtmp.StreamDry")
			case *rtmp.StreamIsRecorded:
				fmt.Println("case *rtmp.StreamIsRecorded")
			case *rtmp.StreamCreatedEvent:
				streamIDs = append(streamIDs, ev.Stream.ID())
				fmt.Println("evt stream created", ev.Stream.ID())
				err = ev.Stream.Play(*streamName, nil, nil, nil)
				if err != nil {
					fmt.Printf("Play error: %s", err.Error())
					os.Exit(-1)
				}
			}
		case <-time.After(5 * time.Second):
			fmt.Println("No data after 5 seconds")
			return
		}
	}
}
