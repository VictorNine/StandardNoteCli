package main

import "os"

func (note *Note) createFile() (string, error) {
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
