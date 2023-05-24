package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type NodePeer struct {
	NodeAddress string `json:"node_address"`
}

// Blockchain represents the blockchain and related operations.
type Blockchain struct {
	Difficulty              int           `json:"difficulty"`
	UnconfirmedTransactions []Transaction `json:"unconfirmed_transactions"`
	Chain                   []Block       `json:"chain"`
	Peers                   []NodePeer    `json:"peers"`
}

// NewBlockchain creates a new blockchain with a genesis block.
func NewBlockchain() (*Blockchain, error) {
	bc := &Blockchain{
		Difficulty: 2,
		Chain:      []Block{},
	}
	err := bc.CreateGenesisBlock()
	if err != nil {
		return nil, err
	}
	return bc, nil
}

// CreateChainFromFile creates a new blockchain by loading the blockchain data from a file.
func CreateChainFromFile(dump string) (*Blockchain, error) {
	// Load the blockchain data from a file
	file, err := os.Open(dump)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blockchain Blockchain
	err = json.NewDecoder(file).Decode(&blockchain)
	if err != nil {
		return nil, err
	}
	return &blockchain, nil
}

// createChainFromDump creates a new blockchain by loading the blockchain data from a dump.
func CreateChainFromDump(chainDump []map[string]interface{}, nodeAddresses []string) (*Blockchain, error) {
	generatedBlockchain, err := NewBlockchain()
	if err != nil {
		return nil, err
	}

	for idx, blockData := range chainDump {
		if idx == 0 {
			continue // Skip genesis block
		}

		block := Block{
			Index:        int(blockData["index"].(float64)),
			Transactions: ParseTransactions(blockData["transactions"].([]interface{})),
			Timestamp:    int64(blockData["timestamp"].(float64)),
			PreviousHash: blockData["previous_hash"].(string),
			Nonce:        int(blockData["nonce"].(float64)),
			Hash:         blockData["hash"].(string),
		}

		err := generatedBlockchain.AddBlock(block)
		if err != nil {
			return nil, err
		}
	}
	nodePeers := []NodePeer{}
	for _, nodeAddress := range nodeAddresses {
		nodePeers = append(nodePeers, NodePeer{NodeAddress: nodeAddress})
	}
	generatedBlockchain.Peers = nodePeers
	return generatedBlockchain, nil
}

// ParseTransactions converts transaction data from interface{} to []Transaction.
func ParseTransactions(transactionsData []interface{}) []Transaction {
	transactions := make([]Transaction, 0, len(transactionsData))

	for _, transactionData := range transactionsData {
		transaction := Transaction{
			Author:    transactionData.(map[string]interface{})["author"].(string),
			Content:   transactionData.(map[string]interface{})["content"].(string),
			Timestamp: int64(transactionData.(map[string]interface{})["timestamp"].(float64)),
		}
		transactions = append(transactions, transaction)
	}

	return transactions
}

// create genesis block
func (bc *Blockchain) CreateGenesisBlock() error {
	genesisBlock := Block{
		Index:        0,
		Transactions: []Transaction{},
		Timestamp:    0,
		PreviousHash: "0",
		Nonce:        0,
	}
	computedHash, err := genesisBlock.ComputeHash()
	if err != nil {
		return err
	}
	genesisBlock.Hash = computedHash
	bc.Chain = append(bc.Chain, genesisBlock)
	return nil
}

// get last block in the chain
func (bc *Blockchain) GetLastBlock() Block {
	return bc.Chain[len(bc.Chain)-1]
}

/**
A function that adds the block to the chain after verification.
Verification includes:
* Checking if the proof is valid.
* The previous_hash referred in the block and the hash of latest block
	in the chain match.
*/
func (bc *Blockchain) AddBlock(block Block) error {

	//compare the previous hash
	if bc.GetLastBlock().Hash != block.PreviousHash {
		return fmt.Errorf("previous hash incorrect")
	}
	//
	if !bc.IsValidProof(block, block.Hash) {
		return fmt.Errorf("block proof invalid")
	}
	bc.Chain = append(bc.Chain, block)
	return nil
}

/**
This function adds the pending transactions to the blockchain
by adding them to the block
and figuring out Proof Of Work.
*/
func (bc *Blockchain) MineBlock() (bool, error) {
	if len(bc.UnconfirmedTransactions) == 0 {
		return false, nil
	}

	lastBlock := bc.GetLastBlock()
	index := lastBlock.Index + 1
	timestamp := time.Now().Unix()
	previousHash := lastBlock.Hash

	newBlock := Block{index, bc.UnconfirmedTransactions, timestamp, previousHash, 0, ""}

	err := bc.ProofOfWork(&newBlock)
	if err != nil {
		return false, err
	}
	bc.AddBlock(newBlock)
	bc.UnconfirmedTransactions = []Transaction{}
	return true, nil
}
func (bc *Blockchain) AddNodePeer(node *NodePeer) {
	bc.Peers = append(bc.Peers, *node)
}
func (bc *Blockchain) AddNewTransaction(transaction *Transaction) {
	bc.UnconfirmedTransactions = append(bc.UnconfirmedTransactions, *transaction)
}

/**
ProofOfWork performs the proof of work algorithm to find a hash
that satisfies the difficulty criteria.
*/
func (bc *Blockchain) ProofOfWork(block *Block) error {
	var (
		computedHash string
		err          error
	)

	computedHash, err = block.ComputeHash()
	if err != nil {
		return err
	}
	for !strings.HasPrefix(computedHash, strings.Repeat("0", bc.Difficulty)) {
		block.Nonce++
		computedHash, err = block.ComputeHash()
		if err != nil {
			return err
		}
	}
	block.Hash = computedHash
	return nil
}

/**
A function to announce to the network once a block has been mined.
    Other blocks can simply verify the proof of work and add it to their
    respective chains.
*/
func (bc *Blockchain) AnnounceNewBlock() {
	for _, peer := range bc.Peers {
		url := peer.NodeAddress + "/add_block"

		blockData, err := json.Marshal(bc.GetLastBlock())
		if err != nil {
			// Handle error
			continue
		}
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(blockData))
		if err != nil {
			log.Printf("Failed to add block to node %s: %v", peer.NodeAddress, err)
			continue
		}
		log.Printf("block added to node %s", peer.NodeAddress)
		defer resp.Body.Close()
	}
}

// perform consensus - If a longer valid chain is
//found, our chain is replaced with it.
func (bc *Blockchain) Consensus() bool {
	currentLen := int64(len(bc.Chain))
	var (
		longestChain  []map[string]interface{}
		newBlockchain *Blockchain
	)

	for _, node := range bc.Peers {
		response, err := http.Get(node.NodeAddress + "/chain")
		if err != nil {
			log.Printf("Failed to get chain from node %s: %v", node.NodeAddress, err)
			continue
		}
		defer response.Body.Close()

		var chainData struct {
			Length int64                    `json:"length"`
			Chain  []map[string]interface{} `json:"chain"`
		}

		err = json.NewDecoder(response.Body).Decode(&chainData)
		if err != nil {
			log.Printf("Failed to decode chain data from node %s: %v", node.NodeAddress, err)
			continue
		}
		newBlockchain, err = CreateChainFromDump(chainData.Chain, []string{})
		if err != nil {
			log.Printf("Failed to create blockchain (%s) from dump: %v", node.NodeAddress, err)
			continue
		}
		if chainData.Length > currentLen && newBlockchain.CheckChainValidity() {
			currentLen = chainData.Length
			longestChain = chainData.Chain
		}
	}

	if longestChain != nil {
		bc.Chain = newBlockchain.Chain
		return true
	}

	return false
}

// IsValidProof checks if the given block hash is a valid proof of work and satisfies the difficulty criteria.
func (bc *Blockchain) IsValidProof(block Block, blockHash string) bool {
	block.Hash = "" // Remove the hash field to recompute the hash
	hash, _ := block.ComputeHash()
	return strings.HasPrefix(blockHash, strings.Repeat("0", bc.Difficulty)) && blockHash == hash
}

// CheckChainValidity checks the validity of the blockchain by verifying each block and its hash.
func (bc *Blockchain) CheckChainValidity() bool {
	previousHash := "0"

	for index, block := range bc.Chain {
		if index != 0 && (!bc.IsValidProof(block, block.Hash) || previousHash != block.PreviousHash) {
			return false
		}
		previousHash = block.Hash
	}

	return true
}
