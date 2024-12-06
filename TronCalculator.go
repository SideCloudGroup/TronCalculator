package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

type Config struct {
	Regex   string `toml:"regex"`
	Num     int    `toml:"num"`
	Threads int    `toml:"threads"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func GenerateKeyPair() (string, string, error) {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %v", err)
	}
	privateKeyBytes := privateKey.D.Bytes()
	privateKeyHex := hex.EncodeToString(privateKeyBytes)
	pubKey := privateKey.PublicKey
	tronAddress := address.PubkeyToAddress(pubKey)
	return privateKeyHex, tronAddress.String(), nil
}

func MatchRegex(regex string, tronAddress string) bool {
	matched, err := regexp.MatchString(regex, tronAddress)
	if err != nil {
		log.Printf("Error matching regex: %v", err)
		return false
	}
	return matched
}

func WriteFile(fileName string, data string) error {
	filePath := fmt.Sprintf("result/%s.txt", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	_, err = file.WriteString(data)
	if err != nil {
		return err
	}
	return nil
}

func worker(config *Config, wg *sync.WaitGroup, mu *sync.Mutex, successCount *int, totalCount *int, stop *bool) {
	defer wg.Done()
	for {
		privateKey, tronAddress, err := GenerateKeyPair()
		if err != nil {
			log.Printf("Error generating key pair: %v", err)
			continue
		}

		if MatchRegex(config.Regex, tronAddress) {
			err := WriteFile(tronAddress, privateKey)
			if err != nil {
				log.Printf("Error writing to file: %v", err)
				continue
			}
			fmt.Printf("Private Key: %s\n", privateKey)
			fmt.Printf("Tron Address: %s\n", tronAddress)

			mu.Lock()
			*successCount++
			if *successCount >= config.Num {
				*stop = true
				mu.Unlock()
				break
			}
			mu.Unlock()
		}

		mu.Lock()
		*totalCount++
		if *stop {
			mu.Unlock()
			return
		}
		if *totalCount%1000000 == 0 {
			fmt.Printf("Attempted %dM times\n", *totalCount)
		}
		mu.Unlock()
	}
}

func main() {
	// Create folder result if not exists
	if _, err := os.Stat("result"); os.IsNotExist(err) {
		err := os.Mkdir("result", 0755)
		if err != nil {
			log.Fatalf("Error creating result directory: %v", err)
		}
	}

	config, err := LoadConfig("config.toml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	totalCount := 0
	stop := false

	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go worker(config, &wg, &mu, &successCount, &totalCount, &stop)
	}
	wg.Wait()
}
