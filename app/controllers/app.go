package controllers

import (
	"code.google.com/p/go.net/websocket"
	"log"
	revelpkg "github.com/revel/revel"
	"hailmary/app/transfer"
	"strconv"
	"strings"
	"time"
)

type App struct {
	*revelpkg.Controller
}

func (c App) Receiver(key string) revelpkg.Result {
	tc := transfer.TheBookKeeper.GetTransferForKey(key)
	return tc.ReadyReceive(c.Controller)
}
func (c App) Sender() revelpkg.Result {
	return c.Render()
}
func (c App) StartReceiver() revelpkg.Result {
	return c.Render()
}

func startTransfer(filename string, numChunks, fsize int64, ws *websocket.Conn) (tc *transfer.TransferConnection) {
	key := transfer.GetKeyForFilename(filename)
	websocket.Message.Send(ws, "key|"+key)
	tc = transfer.TheBookKeeper.GetTransferForKey(key)
	tc.ReadySend(numChunks, fsize, filename)
	websocket.Message.Send(ws, "ready|ready")
	return
}

func (c App) SendChunk(ws *websocket.Conn) revelpkg.Result {
    log.Printf("connected\n")
    filename, numChunks, fsize, err := getFileMetadata(ws)
    if err != nil {
        panic(err)
    }
    tc := startTransfer(filename, numChunks, fsize, ws)
    data := make([]byte, 1048576)
    for {
		startLoopTime := time.Now()
		err := websocket.Message.Receive(ws, &data)
		log.Printf("received msg: %d\n", len(data))
		log.Printf("time since ws recieve start: %f\n ", time.Since(startLoopTime).Seconds())
        if err != nil {
            panic(err)
        }
        if tc == nil {
            panic("also not the correct protocol! no transferConnection available")
        }
        next_chunk := strconv.FormatInt(tc.SendChunk(data), 10)
        if !tc.Finished() {
            websocket.Message.Send(ws, "next|"+next_chunk)
        } else {
            websocket.Message.Send(ws, "end|end")
        }
    }
}

func getFileMetadata(ws *websocket.Conn) (filename string, numChunks int64, fsize int64, theerr error) {
    data := make([]byte, 100)
    for i := 0; i < 3; i++ {
        err := websocket.Message.Receive(ws, &data)
        if err != nil {
            theerr = err
            return
        }
        msg := string(data)
        arr := strings.FieldsFunc(msg, func(c rune) bool {
            return c == '|'
        })
        if arr == nil || len(arr) != 2 {
            panic("protocol was breached!!")
        }
        if arr[0] == "filename" {
            filename = arr[1]
            if numChunks > 0 && fsize > 0 {
                return
            }
        } else if arr[0] == "numChunks" {
            numChunks, err = strconv.ParseInt(arr[1], 10, 32)
            log.Printf("number of chunks: %d\n", numChunks)
            if err != nil {
                theerr = err
                return
            }
            if filename != "" && fsize > 0 {
                return
            }
        } else if arr[0] == "size" {
            fsize, err = strconv.ParseInt(arr[1], 10, 32)
            if err != nil {
                theerr = err
                return
            }
            if filename != "" && numChunks > 0 {
                return
            }
        } else {
            panic("not the correct protocol!")
        }
    }
    panic("should not have reached this block")
}

func (c App) Index() revelpkg.Result {
	return c.Render()
}
