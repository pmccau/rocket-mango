# rocket-mango
### Overview
This is a Discord Bot built using [DiscordGo](https://github.com/bwmarrin/discordgo). It was built on top of the existing [Airhorn example](https://github.com/bwmarrin/discordgo/tree/master/examples/airhorn) and leverages the [DCA file format](https://github.com/bwmarrin/dca).

### Functions
##### Get current commands: !help
Returns complete list of sound bytes

##### Play sound bytes on command: !command
Commands from the !help response will be prepended with a '!'. Each will play a sound byte, which can be added dynamically by channel participants

##### Uploading sound bytes for use on the fly: !newsound
To add sounds to the sound library, upload the sound as an attachment with the comment "!newsound". This will download the attachment, save it, parse to DCA format, then add it to the library with the command as the filename (less extension) prepended with an exclamation mark. If a filename overlaps with an existing command, it will be rejected.
