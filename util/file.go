package util

import (
	"fmt"
	"os"
)

// takes a save path and a variable number of strings and writes them to file separated by new lines
func WriteToFile(savePath string, content ...string) error {
	singleString := ""
	for _, c := range content {
		singleString = fmt.Sprintf("%s \n%s", singleString, c)
	}

	return os.WriteFile(savePath, []byte(singleString), 0644)
}

func AppendToFile(savePath string, content ...string) error {
	f, err := os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	for _, s := range content {
		if _, err = f.WriteString(s + "\n"); err != nil {
			return err
		}
	}
	return nil
}
