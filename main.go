package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"github.com/vinijabes/gocompositor/pkg/compositor"
	"github.com/vinijabes/gocompositor/pkg/compositor/element"
	"github.com/vinijabes/gostreamer/pkg/gstreamer"
)

func main() {
	cmp, err := compositor.NewCompositor()
	if err != nil {
		log.Fatalln(err)
	}

	convert, err := gstreamer.NewElement("videoconvert", "convert")
	if err != nil {
		log.Fatalln(err)
	}

	sink, err := gstreamer.NewElement("autovideosink", "sink")
	if err != nil {
		log.Fatalln(err)
	}

	audioconvert, err := gstreamer.NewElement("audioconvert", "audioconvert")
	if err != nil {
		log.Fatalln(err)
	}

	audiosink, err := gstreamer.NewElement("autoaudiosink", "audiosink")
	if err != nil {
		log.Fatalln(err)
	}

	cmp.Add(convert)
	cmp.Add(sink)
	cmp.Add(audioconvert)
	cmp.Add(audiosink)
	cmp.LinkVideoSink(convert)
	convert.Link(sink)
	cmp.LinkAudioSink(audioconvert)
	audioconvert.Link(audiosink)

	layout := compositor.NewLayout(1280, 720)
	videoRule1 := compositor.NewLayoutRule()
	slot1 := compositor.NewLayoutSlotWithSymetricBorders(0, 0, 640, 360, 320, 180)

	videoRule1.AddSlot(slot1)

	videoRule2 := compositor.NewLayoutRule()
	slot2 := compositor.NewLayoutSlotWithSymetricBorders(0, 0, 640, 360, 0, 180)
	slot3 := compositor.NewLayoutSlotWithSymetricBorders(640, 0, 640, 360, 0, 180)

	videoRule2.AddSlot(slot2)
	videoRule2.AddSlot(slot3)

	videoRule3 := compositor.NewLayoutRule()
	slot4 := compositor.NewLayoutSlot(0, 0, 640, 360)
	slot5 := compositor.NewLayoutSlot(640, 0, 640, 360)
	slot6 := compositor.NewLayoutSlot(320, 360, 640, 360)

	videoRule3.AddSlot(slot4)
	videoRule3.AddSlot(slot5)
	videoRule3.AddSlot(slot6)

	videoRule4 := compositor.NewLayoutRule()
	slot7 := compositor.NewLayoutSlot(0, 0, 640, 360)
	slot8 := compositor.NewLayoutSlot(640, 0, 640, 360)
	slot9 := compositor.NewLayoutSlot(0, 360, 640, 360)
	slot10 := compositor.NewLayoutSlot(640, 360, 640, 360)

	videoRule4.AddSlot(slot7)
	videoRule4.AddSlot(slot8)
	videoRule4.AddSlot(slot9)
	videoRule4.AddSlot(slot10)

	layout.AddRule(videoRule1, 1)
	layout.AddRule(videoRule2, 2)
	layout.AddRule(videoRule3, 3)
	layout.AddRule(videoRule4, 4)
	cmp.SetLayout(layout)

	// Everything below is the pion-WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Allow us to receive 1 audio track, and 2 video tracks
	if _, err = peerConnection.AddTransceiver(webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	var counter int = 0
	// Set a handler for when a new remote track starts, this handler creates a gstreamer pipeline
	// for the given codec

	var mx sync.Mutex
	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		fmt.Println("NEW TRACK RECEIVED")
		mx.Lock()
		// // Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// // This is a temporary fix until we implement incoming RTCP events, then we would push a PLI only when a viewer requests it
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
				if rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		fmt.Println(track.Codec().Name)
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			video, err := element.NewVideoRTC(640, 360, element.VideoRTCCodecVP8)
			if err != nil {
				log.Fatalln(err)
			}

			cmp.AddVideo(video)
			mx.Unlock()

			cmp.Start()

			buf := make([]byte, 1400)
			for {
				i, readErr := track.Read(buf)
				if readErr != nil {
					panic(err)
				}

				video.Push(buf[:i])
			}
		} else {
			audio, err := gstreamer.NewElement("appsrc", fmt.Sprintf("audio_%d", counter))
			if err != nil {
				log.Fatalln(err)
			}

			audio.Set("format", 3)
			audio.Set("is-live", true)
			audio.Set("do-timestamp", true)

			capsfilter, err := gstreamer.NewElement("capsfilter", fmt.Sprintf("audiofilter_%d", counter))
			if err != nil {
				log.Fatalln(err)
			}

			caps, err := gstreamer.NewCapsFromString("application/x-rtp, payload=96, encoding-name=OPUS")
			if err != nil {
				log.Fatalln(err)
			}

			capsfilter.Set("caps", caps)

			depay, err := gstreamer.NewElement("rtpopusdepay", fmt.Sprintf("audiodepay_%d", counter))
			if err != nil {
				log.Fatalln(err)
			}

			dec, err := gstreamer.NewElement("opusdec", fmt.Sprintf("audiodec_%d", counter))
			if err != nil {
				log.Fatalln(err)
			}

			cmp.Add(audio)
			cmp.Add(capsfilter)
			cmp.Add(depay)
			cmp.AddAudio(dec)

			if !audio.Link(capsfilter) ||
				!capsfilter.Link(depay) ||
				!depay.Link(dec) {
				log.Fatalln("Failed to link audio elements")
			}

			counter++
			mx.Unlock()

			fmt.Println("Adding Audio in pipeline")

			time.Sleep(100 * time.Millisecond)
			cmp.Start()
			buf := make([]byte, 1400)
			for {
				i, readErr := track.Read(buf)
				if readErr != nil {
					panic(err)
				}

				audio.Push(buf[:i])
			}
		}

	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	Decode(MustReadStdin(), &offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(Encode(answer))

	// Block forever
	select {}
}
