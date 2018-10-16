package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	sf "github.com/VictorNine/sfgo"
	"github.com/fsnotify/fsnotify"
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

func syncLoop(sess *sf.Session, db *database) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	syncTimer := time.NewTicker(30 * time.Second)
	defer syncTimer.Stop()

	err = watcher.Add("notes")
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				filename := "./" + strings.Replace(event.Name, "\\", "/", -1)
				uuid, ok := files[filename]
				if ok {
					fileToNote(uuid, filename, sess, db)
				}
				log.Println("modified file:", event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)

		case <-syncTimer.C:
			err := sync(sess, db)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

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

func sync(sess *sf.Session, db *database) error {
	unItems, err := db.getUnsyncedItems()
	if err != nil {
		return err
	}

	for _, it := range unItems {
		log.Println(it.UUID + " ready to be synced")
		err := sess.UpdateItem(it)
		if err != nil {
			return err
		}
	}

	// Sync
	items, err := sess.Sync()
	if err != nil {
		return err
	}

	// TODO: Check if there were conflicts before deleting (implement in lib)
	for _, it := range unItems {
		err := db.deletetUnsyncedItem(it.UUID)
		if err != nil {
			return err
		}
	}

	notesSynced := 0 // Counter for notes synced
	for _, item := range items.RetrievedItems {
		if item.ContentType != "Note" {
			continue
		}

		notesSynced++

		// Delete the item from DB if marked for deletion
		if item.Deleted {
			err := db.deleteItem(item.UUID)
			if err != nil {
				return err
			}
			continue
		}

		db.saveItem(&item)

		_, err := createFile(&item, sess)
		if err != nil {
			log.Fatal(err)
		}
	}

	if notesSynced > 0 {
		log.Printf("%v new notes synced to database\n", notesSynced)
	} else {
		log.Println("Database is up to date")
	}

	for _, item := range items.SavedItems {
		if item.ContentType != "Note" {
			continue
		}

		db.saveItem(&item)
	}

	db.setSyncToken(*sess.SyncToken)

	return nil
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

	err = sync(sess, db)
	if err != nil {
		log.Fatal(err)
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
		_, err := createFile(&item, sess)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start sync loop
	syncLoop(sess, db)
}
