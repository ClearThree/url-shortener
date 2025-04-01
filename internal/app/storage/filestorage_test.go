package storage

import (
	"bufio"
	"bytes"
	"github.com/clearthree/url-shortener/internal/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestFileWrapper_Close(t *testing.T) {
	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			err := f.Open()
			assert.NoError(t, err)
			assert.NotNil(t, f.file)
			err = f.Close()
			assert.NoError(t, err)
			assert.Nil(t, f.file)
		})
	}
}

func TestFileWrapper_Create(t *testing.T) {
	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	type args struct {
		id          string
		originalURL string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
	}{
		{
			name: "success",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 0,
			},
			args: args{
				id:          "lelele",
				originalURL: "http://localhost/1",
			},
			want: 1,
		},
		{
			name: "success",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 5,
			},
			args: args{
				id:          "lelele",
				originalURL: "http://localhost/1",
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			got, err := f.Create(tt.args.id, tt.args.originalURL)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "Create(%v, %v)", tt.args.id, tt.args.originalURL)
		})
	}
}

func TestFileWrapper_Open(t *testing.T) {
	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			err := f.Open()
			require.NoError(t, err)
			assert.NotNil(t, f.file)
			assert.NotNil(t, f.writer)
			stat, err := f.file.Stat()
			require.NoError(t, err)
			assert.Equal(t, stat.Mode(), os.FileMode(0644))
		})
	}
}

func TestFileWrapper_ReadNextLine(t *testing.T) {
	var fileMock = bytes.Buffer{}
	fileMock.WriteString(`{"uuid":1,"short_url":"aZdjljjA","original_url":"http://ya.ru"}`)
	fileMock.WriteByte('\n')

	var emptyFileMock = bytes.Buffer{}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0755)
	require.NoError(t, err)

	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	tests := []struct {
		name    string
		fields  fields
		want    *FileRow
		wantErr error
	}{
		{
			name: "success",
			fields: fields{
				file:     devNull,
				reader:   bufio.NewReader(&fileMock),
				writer:   nil,
				lastUUID: 0,
			},
			want: &FileRow{
				UUID:        1,
				ShortURL:    "aZdjljjA",
				OriginalURL: "http://ya.ru",
			},
			wantErr: nil,
		},
		{
			name: "success having empty file",
			fields: fields{
				file:     devNull,
				reader:   bufio.NewReader(&emptyFileMock),
				writer:   nil,
				lastUUID: 0,
			},
			want:    nil,
			wantErr: ErrorFileReadCompletely,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			got, err := f.ReadNextLine()
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equalf(t, tt.want, got, "ReadNextLine()")
			if tt.wantErr == nil {
				assert.Equalf(t, tt.fields.lastUUID+1, f.lastUUID, "ReadNextLine()")
			}
		})
	}
}

func TestFileWrapper_openReadOnly(t *testing.T) {
	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			err := f.openReadOnly()
			require.NoError(t, err)
			assert.NotNil(t, f.file)
			assert.NotNil(t, f.reader)
			stat, err := f.file.Stat()
			require.NoError(t, err)
			assert.Equal(t, stat.Mode(), os.FileMode(0644))
		})
	}
}

func TestFileWrapper_BatchCreate(t *testing.T) {
	type fields struct {
		file     *os.File
		reader   *bufio.Reader
		writer   *bufio.Writer
		lastUUID int32
	}
	type args struct {
		URLs map[string]models.ShortenBatchItemRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
	}{
		{
			name: "success with batch of 2",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 0,
			},
			args: args{
				URLs: map[string]models.ShortenBatchItemRequest{
					"lele": {CorrelationID: "lelele", OriginalURL: "https://ya.ru"},
					"lolo": {CorrelationID: "lololo", OriginalURL: "https://yandex.ru"},
				},
			},
			want: 2,
		},
		{
			name: "success with single URL in batch",
			fields: fields{
				file:     nil,
				reader:   nil,
				writer:   nil,
				lastUUID: 5,
			},
			args: args{
				URLs: map[string]models.ShortenBatchItemRequest{
					"lele": {CorrelationID: "lelele", OriginalURL: "https://ya.ru"},
				},
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileWrapper{
				file:     tt.fields.file,
				reader:   tt.fields.reader,
				writer:   tt.fields.writer,
				lastUUID: tt.fields.lastUUID,
			}
			got, err := f.BatchCreate(tt.args.URLs)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "BatchCreate(%v)", tt.args.URLs)
		})
	}
}
