package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Block struct {
	Index        int           `json:"index"`
	Transactions []Transaction `json:"transactions"`
	Timestamp    int64         `json:"timestamp"`
	PreviousHash string        `json:"previous_hash"`
	Nonce        int           `json:"nonce"`
	Hash         string        `json:"hash"`
}

// Transaction represents a transaction in the blockchain.
type Transaction struct {
	Author    string `json:"author"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

func (bk *Block) ComputeHash() (string, error) {
	bytes, err := json.Marshal(bk)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(bytes)), nil
}
