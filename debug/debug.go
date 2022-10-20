package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type Blob struct {
	Sha256        string
	Blocks        int
	BlocksSize    int
	ContentLenght int
	Origins       []string
}

// func Index(w http.ResponseWriter, r *http.Request) {
// 	url := "http://upload.wikimedia.org/wikipedia/en/b/bc/Wiki.png"

// 	client := &http.Client{
// 		Transport: transport,
// 	}
// 	resp, err := client.Get(url)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer resp.Body.Close()

// 	//copy the relevant headers. If you want to preserve the downloaded file name, extract it with go's url parser.
// 	w.Header().Set("Content-Disposition", "attachment; filename=Wiki.png")
// 	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
// 	w.Header().Set("Content-Length", r.Header.Get("Content-Length"))

// 	//stream the body to the client without fully loading it into memory
// 	io.Copy(w, resp.Body)
// }

// func main() {
// 	http.HandleFunc("/", Index)
// 	err := http.ListenAndServe(":8000", nil)

// 	if err != nil {
// 		fmt.Println(err)
// 	}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		for i := 1; i <= 10; i++ {
			fmt.Fprintf(w, "Chunk #%d\n", i)
			flusher.Flush() // Trigger "chunked" encoding and send a chunk...
			time.Sleep(500 * time.Millisecond)
		}
	})

	log.Print("Listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
