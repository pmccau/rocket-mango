# rocket-mango
![rocket-mango-logo](https://github.com/pmccau/rocket-mango/blob/master/assets/mango.png =150x150)
### Overview
This is a Discord Bot built using [DiscordGo](https://github.com/bwmarrin/discordgo). It was built on top of the existing [Airhorn example](https://github.com/bwmarrin/discordgo/tree/master/examples/airhorn) and leverages the [DCA file format](https://github.com/bwmarrin/dca).

### Functions
##### Get current commands: `!help`
Returns complete list of sound bytes

##### Play sound bytes on command: `!command`
Commands from the !help response will be prepended with a `!`. Each will play a sound byte, which can be added dynamically by channel participants

##### Uploading sound bytes for use on the fly: `!newsound`
To add sounds to the sound library, upload the sound as an attachment with the comment `!newsound`. This will download the attachment, save it, parse to DCA format, then add it to the library with the command as the filename (less extension) prepended with an exclamation mark. If a filename overlaps with an existing command, it will be rejected.

![Example of successful addition of new sound](https://github.com/pmccau/rocket-mango/blob/master/assets/upload-example.PNG)

### Notes

##### How to convert a file to DCA format
1. Make sure you have [ffmpeg](https://www.ffmpeg.org/download.html) installed
2. Download [DCA file format](https://github.com/bwmarrin/dca) or use `dca.exe` from this repo
3. If you downloaded the DCA repo, build it by navigating to the folder and running `go build`. You should now have a dca.exe
4. Run the following command. Replace the `test.m4a` and `test.dca` as needed.`ffmpeg -i test.m4a -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > test.dca`
