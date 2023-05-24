package app

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chokey2nv/ultainfinity/node/blockchain"
	"github.com/go-chi/chi"
)

// Application represents the blockchain application.
type Application struct {
	Blockchain *blockchain.Blockchain
	Router     *chi.Mux
}

const BLOCKCHAIN_FILE = "blockchain.json"

// NewApplication creates a new blockchain application.
func NewApplication() (*Application, error) {
	app := &Application{
		Router: chi.NewRouter(),
	}
	err := app.LoadBlockchain(BLOCKCHAIN_FILE)
	if err != nil {
		return nil, err
	}
	app.SetupRoutes()
	return app, nil
}

// LoadApplication loads the blockchain data from a file and returns a new application instance.
func (app *Application) LoadBlockchain(file string) error {
	var (
		bchain *blockchain.Blockchain
		err    error
	)
	_, err = os.Stat(file)
	if err == nil {
		bchain, err = blockchain.CreateChainFromFile(file)
		if err != nil {
			return err
		}
	} else {
		bchain, err = blockchain.NewBlockchain()
		if err != nil {
			return err
		}
	}
	app.Blockchain = bchain
	return nil
}

// save application (blockchain)
func (app *Application) SaveApplication() error {
	// Save the blockchain data to a file (e.g., JSON)
	file, err := os.Create(BLOCKCHAIN_FILE)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(app.Blockchain)
	if err != nil {
		return err
	}
	return nil
}

// set http routes and handlers
func (app *Application) SetupRoutes() {
	app.Router.Post("/new_transaction", app.HandleNewTransaction)
	app.Router.Get("/chain", app.HandleGetChain)
	app.Router.Get("/mine", app.HandleMine)
	app.Router.Get("/pending_tx", app.HandleGetPendingTransactions)
	app.Router.Post("/add_block", app.HandleVerifyAndAddBlock)
	app.Router.Post("/register_node", app.HandleRegisterNode)
	app.Router.Post("/register_with", app.HandleRegisterNodeWith)
}

// Endpoint /register_with handler function - registers node to list via synced node and syncs the calling node
func (app *Application) HandleRegisterNodeWith(w http.ResponseWriter, r *http.Request) {
	//decode body to get node url for registration
	var node blockchain.NodePeer
	err := json.NewDecoder(r.Body).Decode(&node)
	if err != nil {
		log.Println("Error decoding node:", err)
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}
	// handle empty node address
	if node.NodeAddress == "" {
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}
	// Prepare the request payload
	data := map[string]string{
		"node_address": r.Host,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Make a request to register with the remote node
	response, err := http.Post(node.NodeAddress+"/register_node", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	// Check the response status code
	if response != nil && response.StatusCode == http.StatusOK {

		var responseData struct {
			Chain []map[string]interface{} `json:"chain"`
			Peers []string                 `json:"peers"`
		}
		// decode body (chain as dump)
		err := json.NewDecoder(response.Body).Decode(&responseData)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		//create chain from the received dump
		app.Blockchain, err = blockchain.CreateChainFromDump(responseData.Chain, responseData.Peers)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Registration successful"))
	} else {
		//pass along the response message
		body, _ := ioutil.ReadAll(response.Body)
		w.WriteHeader(response.StatusCode)
		w.Write(body)
	}
}

// Endpoint /register_node handler - adds node peer to list
func (app *Application) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	//decode peer node details
	var node blockchain.NodePeer
	err := json.NewDecoder(r.Body).Decode(&node)
	if err != nil {
		log.Println("Error decoding node:", err)
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}
	//check and prevent empty node_address
	if node.NodeAddress == "" {
		http.Error(w, "Invalid node data", http.StatusBadRequest)
		return
	}

	//add peer to list
	app.Blockchain.AddNodePeer(&node)
	data := map[string]interface{}{
		"chain": app.Blockchain.Chain,
		"peers": app.Blockchain.Peers,
	}
	//marshal blockchain to send back as response data
	bytesBlockchain, err := json.Marshal(data)
	if err != nil {
		log.Println("Error decoding chain:", err)
		http.Error(w, "Invalid chain data", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(bytesBlockchain)
}

// Endpoint /add_block handler - verifies and add new block
func (app *Application) HandleVerifyAndAddBlock(w http.ResponseWriter, r *http.Request) {
	//decode block detail
	var block blockchain.Block
	err := json.NewDecoder(r.Body).Decode(&block)
	if err != nil {
		log.Println("Error decoding block:", err)
		http.Error(w, "Invalid block data", http.StatusBadRequest)
		return
	}
	// add block to chain (verify block)
	err = app.Blockchain.AddBlock(block)
	if err != nil {
		http.Error(w, "Invalid block data", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Success"))
}

//Endpoint /new_transaction handler - Add new/pending/unconfirmed transaction
func (app *Application) HandleNewTransaction(w http.ResponseWriter, r *http.Request) {
	//decode new transaction detail
	var transaction blockchain.Transaction
	err := json.NewDecoder(r.Body).Decode(&transaction)
	if err != nil {
		log.Println("Error decoding transaction:", err)
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}
	//validate transaction details
	if transaction.Author == "" || transaction.Content == "" {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}
	//add timestamp field to the new tx
	transaction.Timestamp = time.Now().Unix()
	//add new tx to pending tx (unconfirmed transactions)
	app.Blockchain.AddNewTransaction(&transaction)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Success"))
}

//Endoing /pending_txs handler - gets pending / unconfirmed transactions
func (app *Application) HandleGetPendingTransactions(w http.ResponseWriter, r *http.Request) {
	//marshal pending transactions and forward as respond data
	responseJSON, err := json.Marshal(app.Blockchain.UnconfirmedTransactions)
	if err != nil {
		log.Println("Error marshaling pending transaction data:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

//Endpoint /chain handler - gets blockchain
func (app *Application) HandleGetChain(w http.ResponseWriter, r *http.Request) {
	//package chian data
	chainData := struct {
		Length     int                `json:"length"`
		Chain      []blockchain.Block `json:"chain"`
		IsValid    bool               `json:"is_valid"`
		Difficulty int                `json:"difficulty"`
		Peers      []string           `json:"peers"`
	}{
		Length:     len(app.Blockchain.Chain),
		Chain:      app.Blockchain.Chain,
		IsValid:    app.Blockchain.CheckChainValidity(),
		Difficulty: app.Blockchain.Difficulty,
		Peers:      []string{}, // Replace with your peers data
	}
	//marshal and forward chain data as http response
	responseJSON, err := json.Marshal(chainData)
	if err != nil {
		log.Println("Error marshaling chain data:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

//Endpoing /mine handler - mines block (pending transactions into a block, then add to chain)
func (app *Application) HandleMine(w http.ResponseWriter, r *http.Request) {
	// mine block
	success, err := app.Blockchain.MineBlock()
	if err != nil {
		log.Println("Error mining block:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	//define response default details
	mineData := struct {
		Message      string                   `json:"message"`
		ChainLength  int                      `json:"chain_length"`
		Transactions []blockchain.Transaction `json:"transactions"`
	}{
		ChainLength: len(app.Blockchain.Chain),
	}
	// if mine is successful add length of txs in block and do consensus and broadcast
	if success {
		chainLength := len(app.Blockchain.Chain) //get chain length before consensus
		app.Blockchain.Consensus()               //persis chain with max length

		if chainLength > len(app.Blockchain.Chain) {
			app.Blockchain.AnnounceNewBlock() // broadcast new block
		}

		// add message and txs in mined block to response data
		mineData.Message = "New block mined"
		mineData.Transactions = app.Blockchain.GetLastBlock().Transactions
	} else {
		mineData.Message = "No transaction to mine"
	}
	//marshall response data and forward
	responseJSON, err := json.Marshal(mineData)
	if err != nil {
		log.Println("Error marshaling chain data:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
