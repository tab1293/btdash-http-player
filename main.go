package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	bencode "github.com/jackpal/bencode-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"io/ioutil"
)

type Segment struct {
	Index     int   `bencode:"index"`
	Start     int64 `bencode:"start"`
	End       int64 `bencode:"end"`
	StartTime int64 `bencode:"start_time"`
	EndTime   int64 `bencode:"end_time"`
}

type Manifest struct {
	Duration int64
	Bitrate  int64
	Segments []Segment
}

type MetaInfo struct {
	Info         InfoDict
	InfoHash     string
	Announce     string
	AnnounceList [][]string `bencode:"announce-list"`
	CreationDate string     `bencode:"creation date"`
	Comment      string
	CreatedBy    string `bencode:"created by"`
	Encoding     string

	Manifest Manifest
}

type FileDict struct {
	Length int64
	Path   []string
	Md5sum string
}

type InfoDict struct {
	PieceLength int64 `bencode:"piece length"`
	Pieces      string
	Private     int64
	Name        string
	// Single File Mode
	Length int64
	Md5sum string
	// Multiple File mode
	Files []FileDict
}

type TorrentService struct {
	client      *torrent.Client
	manifestMap map[string]Manifest
}

func NewTorrentService(c *torrent.Client) *TorrentService {
	var ts TorrentService
	ts.client = c
	ts.manifestMap = make(map[string]Manifest)

	return &ts
}

type PostTorrentRequest struct {
	File string `json:"file"`
}

type PostTorrentResponse struct {
	HexInfoHash string `json:"infoHash"`
}

func PostTorrentHandler(c echo.Context) error {
	r := c.Request().Body
	b, _ := ioutil.ReadAll(r)

	mrs := bytes.NewBuffer(b)
	mr := bytes.NewBuffer(b)

	m := &MetaInfo{}
	err := bencode.Unmarshal(mr, m)
	if err != nil {
		return c.String(400, fmt.Sprintf("Invalid torrent metadata: %s", err))
	}

	mi, err := metainfo.Load(mrs)
	if err != nil {
		return c.String(400, fmt.Sprintf("Invalid torrent passed: %s", err))
	}

	h := mi.HashInfoBytes().HexString()

	ts := c.Get("TorrentService").(*TorrentService)
	t, err := ts.client.AddTorrent(mi)
	if err != nil {
		return c.String(500, fmt.Sprintf("Erroring adding torrent %s", mi.HashInfoBytes().HexString()))
	}

	ts.manifestMap[h] = m.Manifest

	<-t.GotInfo()

	resp := PostTorrentResponse{
		HexInfoHash: t.InfoHash().HexString(),
	}

	return c.JSON(200, resp)
}

func GetTorrentHandler(c echo.Context) error {
	infoHash := c.Param("infohash")

	rangeHeader := c.Request().Header.Get("Range")
	var start, end int64 = -1, -1
	_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
	if err != nil || start < 0 || end < 0 {
		return c.String(400, "Must set valid Range header")
	}

	hash := metainfo.NewHashFromHex(infoHash)
	ts := c.Get("TorrentService").(*TorrentService)
	t, ok := ts.client.Torrent(hash)
	if !ok {
		return c.String(500, fmt.Sprintf("Error getting torrent handle: %s", infoHash))
	}

	r := t.NewReader()
	_, err = r.Seek(start, 0)
	if err != nil {
		return c.String(500, fmt.Sprintf("Error seeking to: %s", start))
	}

	size := end - start + 1
	d := make([]byte, size)
	n, err := r.Read(d)
	if err != nil {
		return c.String(500, fmt.Sprintf("Error reading %s bytes: %s", size))
	}

	fmt.Printf("Read %d bytes\n", n)

	return c.HTMLBlob(200, d)
}

func GetTorrentInfoHandler(c echo.Context) error {
	infoHash := c.Param("infohash")
	hash := metainfo.NewHashFromHex(infoHash)
	ts := c.Get("TorrentService").(*TorrentService)
	t, ok := ts.client.Torrent(hash)
	if !ok {
		return c.String(500, fmt.Sprintf("Error getting torrent handle: %s", infoHash))
	}

	var b bytes.Buffer
	t.Metainfo().Write(&b)
	return c.Blob(200, "application/x-bittorrent", b.Bytes())
}

func GetTorrentManifestHandler(c echo.Context) error {
	infoHash := c.Param("infohash")

	ts := c.Get("TorrentService").(*TorrentService)
	manifest := ts.manifestMap[infoHash]

	return c.JSON(200, manifest)

}

func TorrentServiceMiddleware(ts *TorrentService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("TorrentService", ts)
			return next(c)
		}
	}
}

var Args struct {
	Port int
}

func init() {
	flag.IntVar(&Args.Port, "port", 8080, "Port to run HTTP server on")
}

func main() {
	c, _ := torrent.NewClient(nil)
	ts := NewTorrentService(c)

	e := echo.New()
	e.Use(TorrentServiceMiddleware(ts))
	e.Use(middleware.Static("./static"))

	e.GET("/torrent/:infohash/data", GetTorrentHandler)
	e.GET("/torrent/:infohash/info", GetTorrentInfoHandler)
	e.GET("/torrent/:infohash/manifest", GetTorrentManifestHandler)
	e.POST("/torrent", PostTorrentHandler)
	e.Logger.Fatal(e.Start(":8012"))
}
