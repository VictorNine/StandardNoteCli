package main

import (
	"encoding/json"

	sf "github.com/VictorNine/sfgo"
	"github.com/boltdb/bolt"
)

type database struct {
	conn *bolt.DB
}

func InitDB() (*database, error) {
	db, err := bolt.Open("notes.db", 0600, nil)
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucket([]byte("Notes"))
		return nil
	})

	return &database{conn: db}, err
}

func (db *database) Close() {
	db.conn.Close()
}

func (db *database) setSyncToken(token string) {
	db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Notes"))
		err := b.Put([]byte("synctoken"), []byte(token))
		return err
	})
}

func (db *database) getSyncToken() string {
	var v []byte
	db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Notes"))
		v = b.Get([]byte("synctoken"))
		return nil
	})

	return string(v)
}

func (db *database) saveItem(item *sf.Item) error {
	bytes, err := json.Marshal(item)
	if err != nil {
		return err
	}

	err = db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Notes"))
		err := b.Put([]byte(item.UUID), bytes)
		return err
	})

	return err
}

func (db *database) getItems() ([]sf.Item, error) {
	items := make([]sf.Item, 0)

	err := db.conn.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Notes"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if string(k) == "synctoken" {
				continue
			}
			var item sf.Item
			err := json.Unmarshal(v, &item)
			if err != nil {
				return err
			}
			items = append(items, item)
		}

		return nil
	})

	return items, err
}

func (db *database) deleteItem(uuid string) error {
	err := db.conn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Notes"))
		err := b.Delete([]byte(uuid))
		return err
	})

	return err
}
