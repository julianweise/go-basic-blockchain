package main

// based on https://medium.com/@mycoralhealth/e296282bcffc

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
	"os"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"io"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"fmt"
)

// ##### BLOCK #####

type Block struct {
	Index 		int
	Timestamp 	string
	Data 		string
	Hash 		string
	PrevHash 	string
}

func (block Block) hash() string {
	record := string(block.Index) + block.Timestamp + string(block.Data) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func (block Block) isValid(prevBlock Block) bool {
	if block.Index != prevBlock.Index + 1 {
		return false
	}
	if block.PrevHash != prevBlock.Hash {
		return false
	}
	if block.hash() != block.Hash {
		return false
	}
	return true
}

func createBlock(prevBlock Block, data string) (Block, error) {
	var newBlock Block

	newBlock.Index = prevBlock.Index + 1
	newBlock.Timestamp = time.Now().String()
	newBlock.Data = data
	newBlock.PrevHash = prevBlock.Hash
	newBlock.Hash = newBlock.hash()

	return newBlock, nil
}


// ##### BLOCKCHAIN #####

var Blockchain []Block

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

// ##### WEBSERVER #####

type Message struct {
	Data	string
}

func run() error {
	muxHandler := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        muxHandler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, _ *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	newBlock, err := createBlock(Blockchain[len(Blockchain)-1], m.Data)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}
	if newBlock.isValid(Blockchain[len(Blockchain)-1]) {
		fmt.Println("valid")
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		// prettyPrint into console
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)

}

func respondWithJSON(w http.ResponseWriter, _ *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), "", "", ""}
		genesisBlock.Hash = genesisBlock.hash()
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())

}