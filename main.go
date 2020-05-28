package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var token string
var buffer = make([][]byte, 0)
var validCmds map[string]string
var count int

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord
func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "!help for commands")
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
		buffer = append(buffer, InBuf)
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
	// time.Sleep(250 * time.Millisecond)

	// Start speaking
	vc.Speaking(true)

	//fmt.Printf("Buffer is of size: [%d][%d]")
	// Send buffer data
	//for i, buff := range buffer {
	//	fmt.Printf("[%d]\tLength %d\n", i, len(buff))
	//	vc.OpusSend <- buff
	//}

	for i := count; i < len(buffer); i++ {
		vc.OpusSend <- buffer[i]
	}
	count = len(buffer)

	//vc.OpusSend <- buffer[count]
	//count++

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

func readToken(pathToFile string) string {
	contents, err := ioutil.ReadFile(pathToFile)
	if err != nil {
		panic(err)
	}
	return string(contents)
}


func CheckEncoding(entry string) string {
	pattern := regexp.MustCompile(`\W`)
	splitStr := pattern.Split(entry, -1)
	ext := splitStr[len(splitStr) - 1]
	filename := splitStr[len(splitStr) - 2]

	if ext != "dca" {
		cmd := fmt.Sprintf("ffmpeg -i %s -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > ./sounds/%s.dca", entry, filename)
		fmt.Println("CMD:", cmd)
		_, err  := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			panic(err)
			return "ERROR"
		}
		return fmt.Sprintf("./sounds/%s.dca", filename)
	} else {
		return entry
	}
}

func RegisterCommand(command string, file string) bool {
	if _, ok := validCmds[command]; ok {
		return false
	}

	location := CheckEncoding(file)
	if command[0] != '!' {
		command = fmt.Sprintf("!%s", command)
	}
	validCmds[command] = location
	return true
}

func main() {
	token = readToken("creds.pickle")

	validCmds = make(map[string]string, 0)

	RegisterCommand("!airhorn", "./sounds/airhorn.dca")
	RegisterCommand("!rocketman", "./sounds/rocket_man.dca")
	RegisterCommand("!ROCKETMAN", "./sounds/ROCKETMAN.dca")

	//fmt.Println("Valid cmds", validCmds)

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
	<-sc

	// Close the session
	dg.Close()
}
