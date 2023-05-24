package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
)

// Post represents a single post in the blockchain explorer.
type Post struct {
	Author    string `json:"author"`
	Content   string `json:"content"`
	Index     int    `json:"index"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
}

// ViewData represents the data passed to the template.
type ViewData struct {
	Host        string
	Title       string
	Posts       []Post
	NodeAddress string
}

// Application represents the client application.
type Application struct {
	Router *chi.Mux
	node   string
}

// ConnectedNodeAddress is the address of the connected blockchain node.
const ConnectedNodeAddress = "http://127.0.0.1:8000"

var posts []Post

// NewApplication creates a new blockchain application.
func NewApplication() (*Application, error) {
	app := &Application{
		Router: chi.NewRouter(),
	}
	//check env variable BLOCKCHAIN_NODE for address of the communicating node
	//use default if not found
	if os.Getenv("BLOCKCHAIN_NODE") != "" {
		app.node = "http://" + os.Getenv("BLOCKCHAIN_NODE")
	} else {
		app.node = ConnectedNodeAddress
	}
	//set up api routes
	app.SetupRoutes()
	return app, nil
}
func (app *Application) SetupRoutes() {
	// app.Router.Get("/", app.HandleHomePage)
	app.Router.Get("/", app.IndexHandler)
	app.Router.Post("/submit", app.SubmitTextareaHandler)
}

// Function to fetch the chain from a blockchain node, parse the
// data, and store it locally.
func (app *Application) FetchPosts(posts *[]Post) {
	getChainAddress := app.node + "/chain"
	response, err := http.Get(getChainAddress)
	if err != nil {
		log.Println(err)
	}
	defer response.Body.Close()

	//check request is successful and get response data and pass to posts ref.
	if response != nil && response.StatusCode == http.StatusOK {
		var chain struct {
			Chain []struct {
				Index        int           `json:"index"`
				Transactions []interface{} `json:"transactions"`
				PreviousHash string        `json:"previous_hash"`
			} `json:"chain"`
		}
		if err := json.NewDecoder(response.Body).Decode(&chain); err != nil {
			log.Fatal(err)
		}
		*posts = nil
		for _, block := range chain.Chain {
			for _, tx := range block.Transactions {
				txMap := tx.(map[string]interface{})
				index := block.Index
				hash := block.PreviousHash
				author := txMap["author"].(string)
				content := txMap["content"].(string)
				timestamp := int64(txMap["timestamp"].(float64))

				post := Post{
					Author:    author,
					Content:   content,
					Index:     index,
					Hash:      hash,
					Timestamp: timestamp,
				}
				*posts = append(*posts, post)
			}
		}
	}
}

// Endpoing: Index Request handler to respond with html file
func (app *Application) IndexHandler(w http.ResponseWriter, r *http.Request) {
	//fetch posts from node
	app.FetchPosts(&posts)
	//prepare template for forwarding
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"ReadableTime": app.TimestampToString,
	}).ParseFiles(
		"client/app/templates/index.html",
		"client/app/templates/base.html",
	)
	if err != nil {
		log.Println(err)
	}
	viewData := ViewData{
		Title:       "YourNet: Decentralized content sharing",
		Posts:       posts,
		NodeAddress: ConnectedNodeAddress,
		Host:        r.Host,
	}
	//write template and send with passed in values (struct) and functions
	if err := tmpl.Execute(w, viewData); err != nil {
		log.Println(err)
	}
}

// Endpoint to create a new transaction via our application.
func (app *Application) SubmitTextareaHandler(w http.ResponseWriter, r *http.Request) {
	//collect form body
	postContent := r.FormValue("content")
	author := r.FormValue("author")

	//form struct with the form values
	postObject := struct {
		Author  string `json:"author"`
		Content string `json:"content"`
	}{
		Author:  author,
		Content: postContent,
	}

	//define node public address and path (method)
	newTxAddress := app.node + "/new_transaction"
	payload, err := json.Marshal(postObject)
	if err != nil {
		log.Fatal(err)
	}

	//post new transaction to node and redirect to home page
	_, err = http.Post(newTxAddress, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal(err)
	}

	//redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
//Time stamp to readable string (like - just now, yesterday etc)
func (app *Application) TimestampToString(stamp int64) string {
	timestamp := time.Unix(stamp, 0)
	currentTime := time.Now()
	diff := currentTime.Sub(timestamp)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < 2*time.Minute:
		return "a minute ago"
	case diff < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	case diff < 2*time.Hour:
		return "an hour ago"
	case diff < time.Hour*24:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	case diff < time.Hour*48:
		return "yesterday"
	default:
		return timestamp.Format("January 2, 2006")
	}
}
