package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMemoryRepo_Create(t *testing.T) {
	type args struct {
		id          string
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Successful addition of URL to memory",
			args: args{"lele", "https://ya.ru"},
			want: "lele",
		},
		{
			// The Repository doesn't even have to know what kind of data it stores, so let's check it out
			name: "Successful addition of something to memory",
			args: args{"lele", "something"},
			want: "lele",
		},
		{
			// It also doesn't care about any business logic limitations for keys, values etc.
			name: "Successful addition of something with long ID to memory",
			args: args{"longerThanUsualID", "definitelyNotAnURL"},
			want: "longerThanUsualID",
		},
		{
			// Empty key is also not a problem, even though it's an impossible case
			name: "Empty key success",
			args: args{"", "doesntMatter"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			if got := m.Create(tt.args.id, tt.args.originalURL); got != tt.want {
				t.Errorf("Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryRepo_Read(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		preLoad map[string]string
		want    string
	}{
		{
			name: "Successful read",
			args: args{"lele"},
			preLoad: map[string]string{
				"lele": "https://ya.ru", "lolo": "https://ya.ru", "hehe": "https://vk.com",
			},
			want: "https://ya.ru",
		},
		{
			name:    "Unsuccessful read",
			args:    args{"nonExistent"},
			preLoad: map[string]string{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MemoryRepo{}
			for k, v := range tt.preLoad {
				m.Create(k, v)
			}
			got := m.Read(tt.args.id)
			assert.Equal(t, tt.want, got)
		})
	}
}
