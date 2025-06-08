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

var ErrorFileReadCompletely = errors.New("file has been read completely")

type FileRow struct {
	UUID        int32  `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

type FileWrapper struct {
	file     *os.File
	reader   *bufio.Reader
	writer   *bufio.Writer
	lastUUID int32
}

func (f *FileWrapper) Open() error {
	var err error
	f.file, err = os.OpenFile(config.Settings.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	f.writer = bufio.NewWriter(f.file)
	return nil
}

func (f *FileWrapper) openReadOnly() error {
	var err error
	f.file, err = os.OpenFile(config.Settings.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	f.reader = bufio.NewReader(f.file)
	return nil
}

func (f *FileWrapper) Close() error {
	err := f.writer.Flush()
	if err != nil {
		return err
	}
	fileCloseErr := f.file.Close()
	f.file = nil
	return fileCloseErr
}

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

var FSWrapper = new(FileWrapper)
