package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pmccau/rocket-mango/tools"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

var token string
var buffer = make([][]byte, 0)
var validCmds map[string]string
var count int
var dcaFolder = "sounds/dca"
var stagingFolder = "sounds/staging"
var lock bool // semaphore

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord
func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "!help")
}

// loadSound attempts to load an encoded sound file from disk.
func loadSound(pathToFile string) error {

	file, err := os.Open(pathToFile)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file
		err = binary.Read(file, binary.LittleEndian, &opuslen)
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

	// Start speaking
	vc.Speaking(true)

	// Send the data to the channel
	for i := 0; i < len(buffer); i++ {
		vc.OpusSend <- buffer[i]
	}
	buffer = buffer[len(buffer):]

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
	// Exit if busy
	if lock {
		fmt.Println("EXITING: LOCKED")
		return
	}

	// Ignore messages from the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check for newsound command, download attachment
	if strings.HasPrefix(m.Content, "!newsound") {
		ParseExistingSounds() // Redo the parsing to be sure it's current, no deletions
		for _, att := range m.Attachments {
			splitStr := tools.SplitByNonWord(att.Filename)
			filename := splitStr[len(splitStr) - 2]
			if _, ok := validCmds[filename]; ok {
				content := fmt.Sprintf("!%s is already a command!", filename)
				s.ChannelMessageSend(m.ChannelID, content)
			}

			saveLocation := fmt.Sprintf("%s/%s", stagingFolder, att.Filename)
			tools.DownloadFile(saveLocation, att.URL)
			dcaLocation := tools.ConvertToDCA(saveLocation, dcaFolder)
			RegisterCommand(filename, dcaLocation)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully added !%s", filename))
		}
	}

	// Help message
	if strings.HasPrefix(m.Content, "!help") {
		ParseExistingSounds() // Redo the parsing to be sure it's current, no deletions
		keys := make([]string, 0)
		for k := range validCmds {
			keys = append(keys, k)
		}
		content := "You can ask me to play the following sounds:\n"
		sort.Strings(keys)

		content = fmt.Sprintf("%s\n%s\n\nTo add a new command and sound, click the plus sign to the left of your chat box in the channel, select a sound byte, then add the comment '!newsound'. The new command will be whatever the name of the file is less the extension", content, strings.Join(keys, ", "))
		s.ChannelMessageSend(m.ChannelID, content)
	}

	// Check for valid command
	if val, ok := validCmds[m.Content]; ok {
		ParseExistingSounds() // Redo the parsing to be sure it's current, no deletions
		lock = true

		err := loadSound(val)
		if err != nil {
			lock = false
			panic(err)
		}

		// Find channel
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Couldn't find channel
			lock = false
			panic(err)
			return
		}

		// Find guild for that channel
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Couldn't find guild
			lock = false
			panic(err)
			return
		}

		// Look for message sender in guild's current voice states
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID)
				if err != nil {
					lock = false
					panic(err)
					return
				}
				lock = false
				return
			}
		}
	}
	lock = false
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessage(channel.ID, "rocket-mango is ready. Type !help for more information")
			return
		}
	}
}

// RegisterCommand will add a command to the validCmds map with its associated sound clip
func RegisterCommand(command string, file string) bool {
	if validCmds == nil {
		validCmds = make(map[string]string, 0)
	} else {
		if _, ok := validCmds[command]; ok {
			return false
		}
	}

	location := tools.CheckEncoding(file, dcaFolder)
	if command[0] != '!' {
		command = fmt.Sprintf("!%s", command)
	}
	validCmds[command] = location
	return true
}

// ParseExistingSounds will search the specified dcaFolder for any sound files to be added
// as commands with a prepended !
func ParseExistingSounds() int {
	dcaFiles := tools.GetAllFilesInDir(dcaFolder)
	stagingFiles := tools.GetAllFilesInDir(stagingFolder)
	validCmds = nil

	// Use this map to check whether we already have a given sound as dca
	var dcaFilenames = make(map[string]string, 0)
	for _, file := range dcaFiles {
		splitStr := tools.SplitByNonWord(file)
		filename := splitStr[len(splitStr) - 2]
		dcaFilenames[filename] = file
	}

	// Check for staged sounds that need to be converted. If it's already
	// in there, skip it, otherwise convert
	for _, file := range stagingFiles {
		splitStr := tools.SplitByNonWord(file)
		filename := splitStr[len(splitStr) - 2]
		if _, ok := dcaFilenames[filename]; ok {
			continue
		}
		tools.ConvertToDCA(file, dcaFolder)
	}

	// Refresh the dcaFiles, since we may have added some, then loop
	// through and add them as commands
	dcaFiles = tools.GetAllFilesInDir(dcaFolder)
	for _, file := range dcaFiles {
		splitStr := tools.SplitByNonWord(file)
		filename := splitStr[len(splitStr) - 2]
		RegisterCommand(filename, file)
	}
	return len(dcaFiles)
}

// init does startup tasks for the bot
func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Load in all existing sounds, as well as the token
	ParseExistingSounds()
	var token []byte
	if _, err := os.Stat("creds.pickle"); os.IsNotExist(err) {
		token = []byte(os.Getenv("TOKEN"))
	} else {
		token, err = ioutil.ReadFile("creds.pickle")
		if err != nil {
			panic(err)
		}
	}

	// Initialize the bot and connect to the server
	dg, err := discordgo.New("Bot " + string(token))
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
	fmt.Println("rocket-mango is now running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Close the session
	dg.Close()
}
