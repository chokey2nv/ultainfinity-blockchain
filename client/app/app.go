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
	if os.Getenv("BLOCKCHAIN_NODE") != "" {
		app.node = "http://" + os.Getenv("BLOCKCHAIN_NODE")
	} else {
		app.node = ConnectedNodeAddress
	}
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

	if response.StatusCode == http.StatusOK {
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

func (app *Application) IndexHandler(w http.ResponseWriter, r *http.Request) {
	app.FetchPosts(&posts)
	tmpl, err := template.ParseFiles("client/app/templates/index.html", "client/app/templates/base.html")
	if err != nil {
		log.Println(err)
	}
	// Register the "node_address" function with the template
	tmpl.Funcs(template.FuncMap{
		"ReadableTime": func(time int64) string {
			return app.TimestampToString(time)
		},
	})
	viewData := ViewData{
		Title:       "YourNet: Decentralized content sharing",
		Posts:       posts,
		NodeAddress: ConnectedNodeAddress,
		Host:        r.Host,
	}

	tmpl.Execute(w, viewData)
}

// Endpoint to create a new transaction via our application.
func (app *Application) SubmitTextareaHandler(w http.ResponseWriter, r *http.Request) {
	postContent := r.FormValue("content")
	author := r.FormValue("author")

	postObject := struct {
		Author  string `json:"author"`
		Content string `json:"content"`
	}{
		Author:  author,
		Content: postContent,
	}

	newTxAddress := app.node + "/new_transaction"
	payload, err := json.Marshal(postObject)
	if err != nil {
		log.Fatal(err)
	}

	_, err = http.Post(newTxAddress, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
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
