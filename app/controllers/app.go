package controllers

import (
    revelpkg "github.com/robfig/revel"
    "hailmary/app/transfer"
    "code.google.com/p/go.net/websocket"
    "strings"
    "strconv"
    "fmt"
)

type App struct {
    *revelpkg.Controller
}

func (c App) Receiver(key string) revelpkg.Result {
    tc := transfer.TheBookKeeper.GetTransferForKey(key)
    return tc.ReadyReceive(c.Controller)
}
func(c App) Sender() revelpkg.Result {
    return c.Render()
}
func(c App) StartReceiver() revelpkg.Result {
    return c.Render()
}

func startTransfer(filename string, numChunks, fsize int64, ws *websocket.Conn) (tc *transfer.TransferConnection){
    key := transfer.GetKeyForFilename(filename)
    websocket.Message.Send(ws, "key|" + key)
    tc = transfer.TheBookKeeper.GetTransferForKey(key)
    tc.ReadySend(numChunks, fsize, filename)
    websocket.Message.Send(ws, "ready|ready")
    return
}

func (c App) SendChunk(ws *websocket.Conn) revelpkg.Result {
    fmt.Printf("connected\n")
    websocket.Message.Send(ws, "hi")
    var data []byte
    var filename string
    numChunks := int64(-1)
    fsize := int64(-1)

    var tc *transfer.TransferConnection
    for {
        err := websocket.Message.Receive(ws, &data)
        fmt.Printf("received msg: %d\n", len(data))
        if err != nil {
            fmt.Printf(err.Error())
            return nil
        }
        if len(data) < 100 {
            msg := string(data)

            arr := strings.FieldsFunc(msg, func(c rune) bool {
                return c == '|'
            })
            if arr != nil && len(arr) == 2{
                if arr[0] == "filename" {
                    filename = arr[1]
                    if numChunks > 0 && fsize >0 {
                        tc = startTransfer(filename, numChunks, fsize, ws)
                    }

                } else if arr[0] == "numChunks" {
                    numChunks,err = strconv.ParseInt(arr[1], 10, 32)
                    fmt.Printf("number of chunks: %d\n", numChunks)
                    if filename != "" && fsize > 0{
                        tc = startTransfer(filename, numChunks, fsize, ws)
                    }
                } else if arr[0] == "size" {
                    fsize, err = strconv.ParseInt(arr[1], 10, 32)
                    if filename != "" && numChunks > 0 {
                        tc = startTransfer(filename, numChunks, fsize, ws)
                    }
                }else  {
                    panic("shit")
                }
            } else {
                //if it doesn't match the protocol, just assume its data
                if tc == nil {
                    panic("fuck")
                }
                next_chunk := strconv.FormatInt(tc.SendChunk(data),10)
                if !tc.Finished() {
                    websocket.Message.Send(ws, "next|" + next_chunk)
                } else {
                    websocket.Message.Send(ws, "end|end")
                }
            }

        } else {
            if tc == nil {
                panic("holyballs")
            }
            next_chunk := strconv.FormatInt(tc.SendChunk(data),10)
            if !tc.Finished() {
                websocket.Message.Send(ws, "next|" + next_chunk)
            } else {
                websocket.Message.Send(ws, "end|end")
            }
        }
    }
    return nil
}

func (c App) Index() revelpkg.Result {
    return c.Render()
}

