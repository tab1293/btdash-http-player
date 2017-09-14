package main

import (
	"fmt"
	// "net/http"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type TorrentService struct {
	client *torrent.Client
}

func NewTorrentService(c *torrent.Client) *TorrentService {
	return &TorrentService{
		client: c,
	}
}

type PostTorrentrRequest struct {
	File string `json:"file"`
}

type PostTorrentResponse struct {
	HexInfoHash string `json:"infoHash"`
}

func PostTorrentHandler(c echo.Context) error {
	r := c.Request().Body
	mi, err := metainfo.Load(r)
	if err != nil {
		return c.String(400, "Invalid torrent file passed")
	}

	ts := c.Get("TorrentService").(*TorrentService)
	t, err := ts.client.AddTorrent(mi)
	if err != nil {
		return c.String(500, fmt.Sprintf("Erroring adding torrent %s", mi.HashInfoBytes().HexString()))
	}

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

func TorrentServiceMiddleware(ts *TorrentService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("TorrentService", ts)
			return next(c)
		}
	}
}

func main() {
	c, _ := torrent.NewClient(nil)
	ts := NewTorrentService(c)

	e := echo.New()
	e.Use(TorrentServiceMiddleware(ts))
	e.Use(middleware.Static("./static"))

	e.GET("/torrent/:infohash", GetTorrentHandler)
	e.POST("/torrent", PostTorrentHandler)
	e.Logger.Fatal(e.Start(":8012"))
}
