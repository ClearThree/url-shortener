package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/clearthree/url-shortener/internal/app/config"
	"github.com/clearthree/url-shortener/internal/app/logger"
	"github.com/clearthree/url-shortener/internal/app/models"
)

// ErrorFileReadCompletely is an error that shows that all the file has been read.
var ErrorFileReadCompletely = errors.New("file has been read completely")

// FileRow is a structure that represents the columns of a single object in the file.
type FileRow struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	UUID        int32  `json:"uuid"`
}

// FileWrapper is a structure that wraps all objects required for the file reading and writing.
type FileWrapper struct {
	file     *os.File
	reader   *bufio.Reader
	writer   *bufio.Writer
	lastUUID int32
}

// Open opens the file.
func (f *FileWrapper) Open() error {
	var err error
	f.file, err = os.OpenFile(config.Settings.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	f.writer = bufio.NewWriter(f.file)
	return nil
}

// Open opens the file in read-only mode.
func (f *FileWrapper) openReadOnly() error {
	var err error
	f.file, err = os.OpenFile(config.Settings.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	f.reader = bufio.NewReader(f.file)
	return nil
}

// Close closes the file.
func (f *FileWrapper) Close() error {
	err := f.writer.Flush()
	if err != nil {
		return err
	}
	fileCloseErr := f.file.Close()
	f.file = nil
	return fileCloseErr
}

// Create writes the single row to the file.
func (f *FileWrapper) Create(id string, originalURL string, userID string) (int32, error) {
	if f.file == nil {
		err := f.Open()
		if err != nil {
			return 0, err
		}
	}
	row := FileRow{
		UUID:        f.lastUUID + 1,
		ShortURL:    id,
		OriginalURL: originalURL,
		UserID:      userID,
	}
	data, err := json.Marshal(&row)
	if err != nil {
		return 0, err
	}
	data = append(data, '\n')
	_, err = f.writer.Write(data)
	if err != nil {
		return 0, err
	}
	f.lastUUID++
	err = f.writer.Flush()
	if err != nil {
		return 0, err
	}
	return f.lastUUID, nil
}

// BatchCreate writes multiple rows to the file.
func (f *FileWrapper) BatchCreate(URLs map[string]models.ShortenBatchItemRequest, userID string) (int32, error) {
	if f.file == nil {
		err := f.Open()
		if err != nil {
			return 0, err
		}
	}
	for id, item := range URLs {
		row := FileRow{
			UUID:        f.lastUUID + 1,
			ShortURL:    id,
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		}
		data, err := json.Marshal(&row)
		if err != nil {
			return 0, err
		}
		data = append(data, '\n')
		_, err = f.writer.Write(data)
		if err != nil {
			return 0, err
		}
		f.lastUUID++
	}
	err := f.writer.Flush()
	if err != nil {
		return 0, err
	}
	return f.lastUUID, nil
}

// ReadNextLine reads the next line if exists. Some kind of iterator.
func (f *FileWrapper) ReadNextLine() (*FileRow, error) {
	if f.file == nil {
		err := f.openReadOnly()
		if err != nil {
			return nil, err
		}
	}
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			logger.Log.Debugf("Successfully read storage file, %d lines read", f.lastUUID)
			closeErr := f.file.Close()
			if closeErr != nil {
				return nil, closeErr
			}
			f.file = nil
			return nil, ErrorFileReadCompletely
		}

	}
	fileRow := FileRow{}
	err = json.Unmarshal(data, &fileRow)
	if err != nil {
		return nil, err
	}
	f.lastUUID++

	return &fileRow, nil
}

// FSWrapper is a global variable to use the wrapper in other parts of the program.
var FSWrapper = new(FileWrapper)
