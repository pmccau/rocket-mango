package cmd

import (
	"fmt"
	youtube "github.com/knadh/go-get-youtube/youtube"
)

func GetVideo(endpoint string) {
	// Get the video obj with metadata
	video, err := youtube.Get(endpoint)

	if err != nil {
		panic(err)
	}
	fmt.Println("Found video:", video)

	// Download video and write to file
	option := &youtube.Option{
		Rename: true, // rename file using video title
		Resume: true, // resume cancelled download
		Mp3: true, // extract audio to mp3
	}
	video.Download(0, "video.mp4", option)
}