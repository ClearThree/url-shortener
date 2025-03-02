package service

import (
	"github.com/clearthree/url-shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"

	"testing"
)

type RepoMock struct {
	localStorage map[string]string
}

func (rm RepoMock) Create(id string, originalURL string) string {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
	}
	rm.localStorage[id] = originalURL
	return id
}

func (rm RepoMock) Read(id string) string {
	if rm.localStorage == nil {
		rm.localStorage = make(map[string]string)
	}
	originalURL, ok := rm.localStorage[id]
	if !ok {
		return ""
	}
	return originalURL
}

func TestNewService(t *testing.T) {
	type args struct {
		repo storage.Repository
	}
	tests := []struct {
		name string
		args args
		want ShortURLService
	}{
		{
			name: "Successful creation of service",
			args: args{RepoMock{make(map[string]string)}},
			want: ShortURLService{RepoMock{make(map[string]string)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewService(tt.args.repo))
		})
	}
}

func TestShortURLService_Create(t *testing.T) {
	type fields struct {
		repo storage.Repository
	}
	type args struct {
		originalURL string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Successful creation of short URL",
			fields:  fields{repo: RepoMock{make(map[string]string)}},
			args:    args{originalURL: "https://ya.ru"},
			wantErr: false,
		},
		{
			name:   "Successful creation of short url with long original URL",
			fields: fields{repo: RepoMock{make(map[string]string)}},
			args: args{
				originalURL: "https://example.com/veeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeerylong",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShortURLService{
				repo: tt.fields.repo,
			}
			got, err := s.Create(tt.args.originalURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.IsType(t, "", got)
			assert.NotEmpty(t, got)
			assert.Contains(t, got, hostedOn)
		})
	}
}

func TestShortURLService_Read(t *testing.T) {
	type fields struct {
		repo storage.Repository
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr error
	}{
		{
			name:    "Successful read of short URL",
			fields:  fields{repo: RepoMock{map[string]string{"LElElelE": "https://ya.ru"}}},
			args:    args{id: "LElElelE"},
			want:    "https://ya.ru",
			wantErr: nil,
		},
		{
			name:    "Unsuccessful read of short URL",
			fields:  fields{repo: RepoMock{make(map[string]string)}},
			args:    args{id: "NoNeXiSt"},
			want:    "",
			wantErr: ErrShortURLNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShortURLService{
				repo: tt.fields.repo,
			}
			got, err := s.Read(tt.args.id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_generateID(t *testing.T) {
	tests := []struct {
		name       string
		wantLength int
	}{
		{
			name:       "Successful generation of ID",
			wantLength: shortURLIdLength,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateID()
			assert.Equal(t, tt.wantLength, len(got))

		})
	}
}
