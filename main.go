package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	sf "github.com/VictorNine/sfgo"
)

type Note struct {
	UUID       string                       `json:"-"`
	Text       string                       `json:"text"`
	Title      string                       `json:"title"`
	References []Reference                  `json:"references"`
	AppData    map[string]map[string]string `json:"appData"`
}

type Reference struct {
	UUID        string `json:"uuid"`
	ContentType string `json:"content_type"`
}

var files map[string]string // Map filename to UUID

// Decrypt and unmarshal an item
func itemToNote(sess *sf.Session, item *sf.Item) (Note, error) {
	pt, err := sess.Decrypt(item)
	if err != nil {
		return Note{}, err
	}

	note := Note{}
	err = json.Unmarshal(pt, &note)
	if err != nil {
		return Note{}, err
	}

	note.UUID = item.UUID

	return note, nil
}

func sync(sess *sf.Session, db *database) ([]Note, error) {
	items, err := sess.Sync()
	if err != nil {
		return nil, err
	}

	newNotes := make([]Note, len(items.RetrievedItems))

	for i, item := range items.RetrievedItems {
		if item.ContentType != "Note" {
			continue
		}

		db.saveItem(&item)

		newNotes[i], err = itemToNote(sess, &item)
		if err != nil {
			return nil, err
		}
	}

	for _, item := range items.SavedItems {
		if item.ContentType != "Note" {
			continue
		}

		db.saveItem(&item)
	}

	db.setSyncToken(*sess.SyncToken)

	return newNotes, nil
}

func main() {
	listNotes := flag.Bool("list", false, "Retrieve a list of notes")
	syncAndExit := flag.Bool("sync", false, "Sync database and exit")

	email := flag.String("email", "", "email")
	password := flag.String("password", "", "password")

	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Println("No login information use -email and -password")
		return
	}

	files = make(map[string]string)

	db, err := InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sess := sf.NewSession(
		"https://sync.standardnotes.org",
		*email,
	)

	syncToken := db.getSyncToken()
	if len(syncToken) > 1 {
		sess.SyncToken = &syncToken
	}

	err = sess.Signin(*password)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Login successful!")

	newNotes, err := sync(sess, db)
	if err != nil {
		log.Fatal(err)
	}
	if len(newNotes) > 0 {
		log.Printf("%v new notes synced to database\n", len(newNotes))
	} else {
		log.Println("Database is up to date")
	}

	if *syncAndExit {
		return
	}

	// Get a list of items from the database
	items, err := db.getItems()
	if err != nil {
		log.Fatal(err)
	}

	// List notes
	if *listNotes {
		for _, item := range items {
			note, err := itemToNote(sess, &item)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(note.UUID + " - " + note.Title)
		}

		return // Exit
	}

	err = createDir("notes")
	if err != nil {
		log.Fatal(err)
	}

	// Create txt files from notes
	for _, item := range items {
		note, err := itemToNote(sess, &item)
		if err != nil {
			log.Fatal(err)
		}

		filename, err := note.createFile()
		if err != nil {
			log.Fatal(err)
		}
		files[filename] = note.UUID
	}
}
