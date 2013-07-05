package controllers

import (
    revelpkg "github.com/robfig/revel"
    "filetransfer/app/transfer"
    "code.google.com/p/go.net/websocket"
    "strings"
    "strconv"
    "fmt"
)

type App struct {
    *revelpkg.Controller
}

func (c App) Receiver() revelpkg.Result {
    return transfer.ReadyReceive(c.Controller)
}
func(c App) Sender() revelpkg.Result {
    return c.Render()
}

func (c App) SendChunk(ws *websocket.Conn) revelpkg.Result {
    var data []byte
    var filename string
    numChunks := int64(-1)
    for {
        err := websocket.Message.Receive(ws, &data)
        if err != nil {
            panic("wut")
        }
        if len(data) < 100 {
            msg := string(data)
            fmt.Printf("in string: %s\n", msg)

            arr := strings.FieldsFunc(msg, func(c rune) bool {
                return c == '|'
            })
            if arr != nil && len(arr) == 2{
                if arr[0] == "filename" {
                    filename = arr[1]
                    if numChunks > 0 {
                        transfer.ReadySend(numChunks, filename)
                        websocket.Message.Send(ws, "ready")
                    }

                } else if arr[0] == "numChunks" {
                    numChunks,err = strconv.ParseInt(arr[1], 10, 32)
                    if filename != "" {
                        transfer.ReadySend(numChunks, filename)
                        websocket.Message.Send(ws, "ready")
                    }
                } else  {
                    panic("shit")
                }
            }

        } else {
            transfer.SendChunk(data)
        }
    }
}

func (c App) Index() revelpkg.Result {
    return c.Render()
}

