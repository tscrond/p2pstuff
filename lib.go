package main

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func savePrivKey(nodeInfo NodeInfo, fileName string) error {

	privBytes, err := nodeInfo.Privkey.Raw()
	if err != nil {
		return err
	}

	// Write raw bytes to a binary file
	err = os.WriteFile(fileName, privBytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key to file: %v", err)
	}

	log.Println("Private key saved to", fileName)

	return nil
}

func readNodeInfoFromFile(fileName string) (*NodeInfo, error) {

	privBytes, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	privKey := ed25519.PrivateKey(privBytes)

	libp2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %v", err)
	}

	nodeinfo := NewNodeInfo(libp2pPrivKey)

	return nodeinfo, nil
}

func ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
