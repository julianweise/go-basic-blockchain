package main

// based on https://medium.com/@mycoralhealth/e296282bcffc

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
	"log"
	"github.com/joho/godotenv"
	"net"
	"os"
	"io"
	"bufio"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
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

// ##### NODE #####

var bcServer chan []Block

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	bcServer = make(chan []Block)
	genesisBlock := Block{0, time.Now().String(), "", "", ""}
	spew.Dump(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	server, err := net.Listen("tcp", ":" + os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	for {
		connection, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(connection)
	}
}

func handleConnection(connection net.Conn) {
	defer connection.Close()

	io.WriteString(connection, "\nEnter some textual data to be added to the Blockchain: ")
	scanner := bufio.NewScanner(connection)

	go func() {
		for scanner.Scan() {
			newBlock, err := createBlock(Blockchain[len(Blockchain) - 1], scanner.Text())
			if err != nil {
				log.Println(err)
				continue
			}
			if newBlock.isValid(Blockchain[len(Blockchain)-1]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
			}

			bcServer <- Blockchain
			io.WriteString(connection, "\nEnter a new string: ")
		}
	}()

	go func() {
		for {
			time.Sleep(30 * time.Second)
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(connection, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}

}