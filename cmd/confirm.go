package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func confirmAnswer(reader *os.File) error {
	line, err := readLine(reader)
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "y" || line == "yes" {
		return nil
	}
	return errors.New("operation cancelled")
}

func readLine(reader *os.File) (string, error) {
	bufReader := bufio.NewReader(reader)
	line, err := bufReader.ReadString('\n')
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return "", err
		}
	}
	return line, nil
}
