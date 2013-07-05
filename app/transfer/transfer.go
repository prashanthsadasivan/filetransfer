package transfer

import (
    revelpkg "github.com/robfig/revel"
    "io"
    "time"
)

var (
    sharedStream = make(chan byte)
    filenamePipe = make(chan string, 1)
    readerReady = make(chan bool, 1)
    totalNumberOfChunks int64
    numChunksSent = int64(0)
)

type StreamReader struct {
    New chan byte
}

func (sr *StreamReader) Read(p []byte) (n int, err error) {
    lengthAsked := len(p)
    n = 0;
    err = nil
    for n < lengthAsked {
        thebyte, ok := <-sr.New
        if !ok {
            return n, io.EOF
            break;
        }
        p[n] = thebyte
        n++
    }
    return
}

// Return a file, either displayed inline or downloaded as an attachment.
// The name and size are taken from the file info.
func RenderBS(c *revelpkg.Controller, filename string) revelpkg.Result {
    var (
        modtime       = time.Now()
    )
    readerReady <- true
    sr := new(StreamReader)
    sr.New = sharedStream
    c.Response.Out.Header().Set("Content-Type", "application/octet-stream")
    return &revelpkg.BinaryResult{
        Reader:   sr,
        Name:     filename,
        Delivery: "attachment",
        Length:   -1, // http.ServeContent gets the length itself
        ModTime:  modtime,
    }
}

func ReadyReceive(c *revelpkg.Controller) revelpkg.Result {
    filename := <-filenamePipe
    return RenderBS(c, filename)
}

func ReadySend(numChunks int64, filename string) bool{
    totalNumberOfChunks = numChunks
    filenamePipe <- filename
    return <- readerReady
}

func SendChunk(chunk []byte) {
    if(numChunksSent < totalNumberOfChunks) {
        for _,element := range chunk{
            sharedStream <- element
        }
        numChunksSent++;
        if numChunksSent == totalNumberOfChunks {
            close(sharedStream)
        }
    } else {
        panic("wtf")
    }
}

func init() {
}

