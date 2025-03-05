# Spotydw
Spotydw is a CLI tool to download tracks, albums or playlists from Spotify
## Prerequisites
- [Ffmpeg](https://ffmpeg.org)
- [Yt-dlp](https://github.com/yt-dlp/yt-dlp)
## Usage
- To get a Spotify client ID and secret, go to [Spotify for Developers](https://developer.spotify.com/dashboard/applications) and create an app.
- In the .env file:
```env
SPOTIFY_ID=<your_client_id>
SPOTIFY_SECRET=<your_secret>
```
Then you can run spotydw:
```bash
Usage:
  spotydw [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  download    Download a track, album, or playlist from Spotify
  help        Help about any command

Flags:
  -h, --help   help for spotydw

Use "spotydw [command] --help" for more information about a command.
```
