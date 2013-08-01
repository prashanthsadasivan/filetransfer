package transfer

import (
    revelpkg "github.com/robfig/revel"
    "io"
    "time"
    "sync"
    "fmt"
    "crypto/md5"
)

type TransferConnection struct {
    sharedStream chan byte
    filenamePipe chan string
    readerReady chan bool
    totalNumberOfChunks int64
    numChunksSent int64
    key string
}

type BookKeeper struct {
    connections map[string] *TransferConnection
    mutex sync.Mutex
}

func (bk *BookKeeper) GetTransferForKey(key string) *TransferConnection {
    bk.mutex.Lock()
    defer bk.mutex.Unlock()
    tc :=  bk.connections[key]
    if tc == nil {
        fmt.Printf("making tc for key: %s\n", key)
        tc = new(TransferConnection)
        tc.sharedStream = make(chan byte)
        tc.filenamePipe = make(chan string, 1)
        tc.readerReady = make(chan bool, 1)
        tc.totalNumberOfChunks = int64(0)
        tc.numChunksSent = int64(0)
        bk.connections[key] = tc
    }
    return tc
}


func (bk *BookKeeper) DeleteTransferForKey(key string) {
    bk.mutex.Lock()
    defer bk.mutex.Unlock()
    delete(bk.connections, key)
    return
}

var (
    TheBookKeeper BookKeeper
)

type StreamReader struct {
    New chan byte
}

func (sr *StreamReader) Read(p []byte) (n int, err error) {
    lengthAsked := len(p)
    n = 0;
    err = nil
    for n < lengthAsked {
        fmt.Printf("waiting for bytes\n")
        thebyte, done := <-sr.New
        if !done {
            fmt.Printf("no more bytes\n")
            return n, io.EOF
        }
        p[n] = thebyte
        n++
    }
    fmt.Printf("reading %d bytes\n", n)
    return
}

// Return a file, either displayed inline or downloaded as an attachment.
// The name and size are taken from the file info.
func (tc *TransferConnection) RenderBS(c *revelpkg.Controller, filename string) revelpkg.Result {
    var (
        modtime       = time.Now()
    )
    tc.readerReady <- true
    sr := new(StreamReader)
    sr.New = tc.sharedStream
    c.Response.Out.Header().Set("Content-Type", "application/octet-stream")
    return &revelpkg.BinaryResult{
        Reader:   sr,
        Name:     filename,
        Delivery: "attachment",
        Length:   tc.totalNumberOfChunks*131072, // http.ServeContent gets the length itself
        ModTime:  modtime,
    }
}

func (tc *TransferConnection) ReadyReceive(c *revelpkg.Controller) revelpkg.Result {
    tc.key = <-tc.filenamePipe
    return tc.RenderBS(c, tc.key)
}
func GetKeyForFilename(filename string) string {
    h := md5.New()
    io.WriteString(h, filename)
    return fmt.Sprintf("%x", h.Sum(nil))
}

func (tc *TransferConnection) ReadySend(numChunks int64, filename string) bool{
    tc.totalNumberOfChunks = numChunks
    tc.filenamePipe <- filename
    return <- tc.readerReady
}

func (tc *TransferConnection) SendChunk(chunk []byte) int64{
    if(tc.numChunksSent < tc.totalNumberOfChunks) {
        for _,element := range chunk{
            tc.sharedStream <- element
        }
        tc.numChunksSent++;
        if tc.numChunksSent == tc.totalNumberOfChunks {
            close(tc.sharedStream)
            TheBookKeeper.DeleteTransferForKey(tc.key)
        }
        return tc.numChunksSent
    } else {
        panic("wtf")
    }
}

func (tc *TransferConnection) Finished() bool {
    return tc.numChunksSent == tc.totalNumberOfChunks
}

func init() {
    TheBookKeeper.connections = make(map[string]*TransferConnection)
}


