# btdash-http-player
This is a btdash playback implementation that makes use of an HTTP server to forward BitTorrent traffic to a web browser's 
MediaSource API. 

## Usage
`./btdash-http-player -port 8888`

- Open https://localhost:8888/index.html in your web browser
- Use the file selector on the page to load a torrent file packaged with [btdash-packager](https://github.vimeows.com/thomas/btdash-packager)
- Watch the video playback :tv:

#### GET Routes
- `/torrent/<infohash>/data` is used in conjuction with the Range request header to get the video file data
- `/torrent/<infohash>/info` returns the torrent file for download
- `/torrent/<infohash>/manifest` returns the playback manifest embed in the torrent file

### POST Route
- `/torrent` Adds the torrent file supplied in the request body



