package proxy

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"plusvasis/internal/controllers/nomad"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/labstack/echo/v4"
)

type NomadProxyController struct {
	Client nomad.NomadClient
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const idleTimeout = 30 * time.Second

func (n *NomadProxyController) AllocExec(c echo.Context) error {
	id := c.Param("id")
	command := c.QueryParam("command")
	if command == "" {
		return fmt.Errorf("missing query parameters")
	}

	alloc, err := n.parseRunningAlloc(id)
	if err != nil {
		return err
	}

	baseURL := "wss://nomad.local.cawnj.dev/"
	path := "v1/client/allocation/" + alloc.ID + "/exec"
	queryParams := url.Values{}
	queryParams.Add("command", command)
	queryParams.Add("task", alloc.TaskGroup)
	queryParams.Add("tty", "true")
	queryParams.Add("ws_handshake", "true")

	url := baseURL + path + "?" + queryParams.Encode()

	nomadConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	defer nomadConn.Close()

	clientConn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	err = nomadConn.SetReadDeadline(time.Now().Add(idleTimeout))
	if err != nil {
		return err
	}
	err = clientConn.SetReadDeadline(time.Now().Add(idleTimeout))
	if err != nil {
		return err
	}

	log.Printf("Started terminal session for job %s", id)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		n.forwardMessages(clientConn, nomadConn)
		wg.Done()
	}()
	go func() {
		n.forwardMessages(nomadConn, clientConn)
		wg.Done()
	}()
	wg.Wait()

	log.Printf("Stopped terminal session for job %s", id)
	return nil
}

func (n *NomadProxyController) forwardMessages(srcConn, dstConn *websocket.Conn) {
	for {
		msgType, msg, err := srcConn.ReadMessage()
		if err != nil {
			break
		}
		err = dstConn.WriteMessage(msgType, msg)
		if err != nil {
			break
		}
		err = srcConn.SetReadDeadline(time.Now().Add(idleTimeout))
		if err != nil {
			break
		}
	}
}

func (n *NomadProxyController) parseRunningAlloc(jobId string) (*structs.AllocListStub, error) {
	data, err := n.Client.Get(fmt.Sprintf("/job/%s/allocations", jobId))
	if err != nil {
		return nil, err
	}

	var allocs []structs.AllocListStub
	err = json.Unmarshal(data, &allocs)
	if err != nil {
		return nil, err
	}
	for _, alloc := range allocs {
		if alloc.ClientStatus == "running" || alloc.ClientStatus == "pending" {
			return &alloc, nil
		}
	}

	return nil, fmt.Errorf("no running alloc found for job %s", jobId)
}

func (n *NomadProxyController) StreamLogs(c echo.Context) error {
	id := c.Param("id")
	task := c.QueryParam("task")
	logType := c.QueryParam("type")
	if task == "" || logType == "" {
		return fmt.Errorf("missing query parameters")
	}

	alloc, err := n.parseRunningAlloc(id)
	if err != nil {
		return err
	}

	baseURL := "https://nomad.local.cawnj.dev/"
	path := "v1/client/fs/logs/" + alloc.ID
	queryParams := url.Values{}
	queryParams.Add("task", task)
	queryParams.Add("type", logType)
	queryParams.Add("follow", "true")
	queryParams.Add("offset", "50000")
	queryParams.Add("origin", "end")

	url := baseURL + path + "?" + queryParams.Encode()
	resp, err := n.forwardRequest(c, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("Started streaming %s for job %s", logType, id)

	err = streamResponse(c, resp)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	log.Printf("Stopped streaming %s for job %s", logType, id)
	return nil
}

func (n *NomadProxyController) forwardRequest(c echo.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(c.Request().Method, url, c.Request().Body)
	if err != nil {
		return nil, err
	}

	for key, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	return resp, err
}

func streamResponse(c echo.Context, resp *http.Response) error {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		resp.Body = gzipReader
	}

	buf := make([]byte, 8192)
	for {
		bytesRead, err := resp.Body.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		select {
		case <-c.Request().Context().Done():
			return nil
		default:
		}

		_, err = c.Response().Writer.Write(buf[:bytesRead])
		if err != nil {
			return err
		}

		if flusher, ok := c.Response().Writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	return nil
}