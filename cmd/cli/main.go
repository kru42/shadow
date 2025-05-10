package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/kru42/shadow/internal/encryption"
)

var (
	relayAddr = flag.String("relay", "localhost:9999", "relay address")
	meID      = flag.String("id", "", "your peer ID (hex of public key)")
)

func main() {
	flag.Parse()

	// load or generate our keypair
	var pub, priv *[32]byte
	var err error
	if *meID == "" {
		pub, priv, err = encryption.GenerateKeyPair()
		if err != nil {
			log.Fatal(err)
		}
		*meID = hex.EncodeToString(pub[:])
		fmt.Println("Generated your ID:", *meID)
	} else {
		raw, err := hex.DecodeString(*meID)
		if err != nil || len(raw) != 32 {
			log.Fatal("invalid id")
		}
		pubArray := [32]byte{}
		copy(pubArray[:], raw)
		pub = &pubArray
		priv = nil // only decrypt when fetching: you'll need to load your private key similarly
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("cmd> ")
		line, _ := reader.ReadString('\n')
		switch {
		case strings.HasPrefix(line, "send "):
			// send <destID> <message>
			parts := strings.SplitN(line, " ", 3)
			destRaw := parts[1]
			msg := strings.TrimRight(parts[2], "\n")

			destBytes, err := hex.DecodeString(destRaw)
			if err != nil || len(destBytes) != 32 {
				fmt.Println("invalid destID")
				continue
			}
			var destPub [32]byte
			copy(destPub[:], destBytes)

			blob, err := encryption.Encrypt([]byte(msg), &destPub, priv)
			if err != nil {
				fmt.Println("encrypt:", err)
				continue
			}

			c, err := net.Dial("tcp", *relayAddr)
			if err != nil {
				fmt.Println("dial:", err)
				continue
			}
			fmt.Fprintf(c, "SEND\n%s\n%s\n", destRaw, string(blob))
			resp, _ := bufio.NewReader(c).ReadString('\n')
			fmt.Println("→", strings.TrimSpace(resp))
			c.Close()

		case strings.HasPrefix(line, "fetch"):
			c, err := net.Dial("tcp", *relayAddr)
			if err != nil {
				fmt.Println("dial:", err)
				continue
			}
			fmt.Fprintf(c, "FETCH\n%s\n", *meID)
			r := bufio.NewReader(c)
			for {
				l, _ := r.ReadString('\n')
				if strings.TrimSpace(l) == "END" {
					break
				}
				blob := []byte(strings.TrimRight(l, "\n"))
				// here you'd decrypt with the sender's public key + your priv
				fmt.Printf("📨 raw: %x\n", blob)
			}
			c.Close()

		case strings.HasPrefix(line, "exit"):
			return

		default:
			fmt.Println("commands: send <destID> <msg>, fetch, exit")
		}
	}
}
