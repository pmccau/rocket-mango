package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var token string
var buffer = make(*[][]byte, 0)
var validCmds map[string]string

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord
func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "!airhorn")
}

// loadSound attempts to load an encoded sound file from disk.
func loadSound(pathToFile string) error {

	fmt.Println("Loading sound from", pathToFile)
	file, err := os.Open(pathToFile)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file
		err = binary.Read(file, binary.LittleEndian, &opuslen)
		fmt.Println(err)
		fmt.Println(opuslen)
		// If end of file, return
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		}

		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("Read in", opuslen, "bytes")
		// Read encoded pcm from dca file
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		if err != nil {
			fmt.Println(err)
			return err
		}
		newBuff := append(*buffer, InBuf)
		buffer = &newBuff
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string) (err error) {

	// Join the voice channel
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Sleep for this amount of time before playing sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking
	vc.Speaking(true)

	//fmt.Printf("Buffer is of size: [%d][%d]")
	// Send buffer data
	for _, buff := range *buffer {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for this amount of time before ending
	time.Sleep(250 * time.Millisecond)

	// Disconnect from provided voice channel
	vc.Disconnect()

	return nil
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check for regex
	for k, v := range validCmds {
		if strings.HasPrefix(m.Content, k) {
			err := loadSound(v)
			if err != nil {
				panic(err)
			}

			// Find channel
			c, err := s.State.Channel(m.ChannelID)
			if err != nil {
				// Couldn't find channel
				panic(err)
				return
			}

			// Find guild for that channel
			g, err := s.State.Guild(c.GuildID)
			if err != nil {
				// Couldn't find guild
				panic(err)
				return
			}

			// Look for message sender in guild's current voice states
			for _, vs := range g.VoiceStates {
				if vs.UserID == m.Author.ID {
					err = playSound(s, g.ID, vs.ChannelID)
					if err != nil {
						panic(err)
						return
					}
					return
				}
			}
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessage(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channelt o play a sound")
			return
		}
	}
}

// init does startup tasks for the bot
func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	token = "NzE0MTcwMzQ2Nzg5NDA0Nzc0.Xsq1Kg.t7Ts_Hdp1HZ2A4fxHSXXNA7oGGU"
	if token == "" {
		fmt.Println("Please provide a token with -t")
		return
	}

	validCmds = make(map[string]string, 0)
	validCmds["!airhorn"] = "./sounds/airhorn.dca"
	validCmds["!rocketman"] = "./sounds/rocketman0.dca"
	validCmds["!ROCKETMAN"] = "./sounds/rocketman1.dca"

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	// Register ready as callback for ready events
	dg.AddHandler(ready)

	// Register messageCreate as callback for messageCreate events
	dg.AddHandler(messageCreate)

	// Register guildCreate as callback for guildCreate events
	dg.AddHandler(guildCreate)

	// Open websocket, begin listening
	err = dg.Open()
	if err != nil {
		panic(err)
	}

	// Wait here until killed
	fmt.Println("Airhorn now running.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<- sc

	// Close the session
	dg.Close()
}