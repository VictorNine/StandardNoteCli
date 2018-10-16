package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	sf "github.com/VictorNine/sfgo"
)

func createFile(item *sf.Item, sess *sf.Session) (string, error) {
	// TODO: Check if the file exists in map. rename to Title+"1"
	note, err := itemToNote(sess, item)
	if err != nil {
		return "", err
	}

	filename := "./notes/" + note.Title + ".txt"
	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.WriteString(note.Text)
	if err != nil {
		return "", err
	}

	f.Sync()

	files[filename] = item.UUID

	return filename, nil
}

// Create dir if it doesent exist
func createDir(name string) error {
	_, err := os.Stat(name)

	if os.IsNotExist(err) {
		err = os.MkdirAll(name, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// Encrypt file content and put it in DB as unsynced
func fileToNote(uuid, filename string, sess *sf.Session, db *database) {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	item, err := db.getItem(uuid)
	if err != nil {
		log.Fatal(err)
	}

	note, err := itemToNote(sess, &item)
	if err != nil {
		log.Fatal(err)
	}

	note.Text = string(text)
	bNote, err := json.Marshal(&note)
	if err != nil {
		log.Fatal(err)
	}

	item.PlanText = bNote

	err = sess.EncryptItem(&item)
	if err != nil {
		log.Fatal(err)
	}

	err = db.newUnsyncedItem(&item)
	if err != nil {
		log.Fatal(err)
	}
}
