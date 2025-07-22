package config

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpAddress_Set(t *testing.T) {
	type fields struct {
		Scheme string
		Host   string
		Port   int
	}
	type args struct {
		flagValue string
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "set http address success",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			args: args{
				flagValue: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "set http address success with trailing slash",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			args: args{
				flagValue: "http://localhost:8080/",
			},
			wantErr: false,
		},
		{
			name: "set http address fail - missing port",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			args: args{
				flagValue: "http://localhost8080/",
			},
			wantErr: true,
		},
		{
			name: "set http address fail - missing scheme",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			args: args{
				flagValue: "http:/localhost:8080/",
			},
			wantErr: true,
		},
		{
			name: "set http address fail - unprocessable address",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			args: args{
				flagValue: "httplocalhost8080",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HTTPAddress{}
			correctAddress := HTTPAddress{
				Scheme: tt.fields.Scheme,
				Host:   tt.fields.Host,
				Port:   tt.fields.Port,
			}
			if err := h.Set(tt.args.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, correctAddress, h)
			}
		})
	}
}

func TestHttpAddress_String(t *testing.T) {
	type fields struct {
		Scheme string
		Host   string
		Port   int
	}
	tests := []struct {
		name   string
		want   string
		fields fields
	}{
		{
			name: "craft http address string success",
			fields: fields{
				Scheme: "http://",
				Host:   "localhost",
				Port:   8080,
			},
			want: "http://localhost:8080/",
		},
		{
			name: "craft http address string success",
			fields: fields{
				Scheme: "https://",
				Host:   "127.0.0.1",
				Port:   8089,
			},
			want: "https://127.0.0.1:8089/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTPAddress{
				Scheme: tt.fields.Scheme,
				Host:   tt.fields.Host,
				Port:   tt.fields.Port,
			}
			assert.Equal(t, tt.want, h.String())
		})
	}
}

func TestNetAddress_Set(t *testing.T) {
	type fields struct {
		Host string
		Port int
	}
	type args struct {
		flagValue string
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "set net address success",
			fields: fields{
				Host: "localhost",
				Port: 8080,
			},
			args: args{
				flagValue: "localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "set net address success with trailing slash",
			fields: fields{
				Host: "localhost",
				Port: 8080,
			},
			args: args{
				flagValue: "localhost:8080/",
			},
			wantErr: false,
		},
		{
			name: "set net address fail - missing port",
			fields: fields{
				Host: "localhost",
				Port: 8080,
			},
			args: args{
				flagValue: "localhost8080/",
			},
			wantErr: true,
		},
		{
			name: "set net address fail - unprocessable address",
			fields: fields{
				Host: "localhost",
				Port: 8080,
			},
			args: args{
				flagValue: "localhost8080",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NetAddress{
				Host: tt.fields.Host,
				Port: tt.fields.Port,
			}
			if err := n.Set(tt.args.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNetAddress_String(t *testing.T) {
	type fields struct {
		Host string
		Port int
	}
	tests := []struct {
		name   string
		want   string
		fields fields
	}{
		{
			name: "craft http address string success",
			fields: fields{
				Host: "localhost",
				Port: 8080,
			},
			want: "localhost:8080",
		},
		{
			name: "craft http address string success",
			fields: fields{
				Host: "127.0.0.1",
				Port: 8089,
			},
			want: "127.0.0.1:8089",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NetAddress{
				Host: tt.fields.Host,
				Port: tt.fields.Port,
			}
			assert.Equal(t, tt.want, n.String())
		})
	}
}

func TestParseFlags(t *testing.T) {
	test := struct {
		wantAddress     string
		wantBaseAddress string
		flags           []string
	}{
		flags: []string{
			"lel", "-a=localhost:8083", "-b=http://localhost:8083",
		},
		wantAddress:     "localhost:8083",
		wantBaseAddress: "http://localhost:8083/",
	}
	os.Args = test.flags
	ParseFlags()
	assert.Equal(t, argsConfig.Address.String(), test.wantAddress)
	assert.Equal(t, argsConfig.HostedOn.String(), test.wantBaseAddress)
}

func TestFileStoragePath_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "set file storage path success",
			args: args{
				s: "./",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileStoragePath{}
			err := f.Set(tt.args.s)
			require.NoError(t, err)
			assert.Equal(t, tt.args.s, f.Path)
		})
	}
}

func TestFileStoragePath_String(t *testing.T) {
	type fields struct {
		Path string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "get file storage path success",
			fields: fields{
				Path: "./",
			},
			want: "./",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileStoragePath{
				Path: tt.fields.Path,
			}
			assert.Equalf(t, tt.want, f.String(), "String()")
		})
	}
}

func TestGetOrCreateCertAndKey(t *testing.T) {
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		name    string
		exist   bool
	}{
		{
			name:    "get file - no files exist",
			exist:   false,
			wantErr: assert.NoError,
		},
		{
			name:    "get file - files exist",
			exist:   true,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.exist {
				Settings.CertPath = "./cert.pem"
				Settings.KeyPath = "./key.pem"
				_, _, err := generateCertAndKey()
				assert.NoError(t, err)
			} else {
				Settings.CertPath = "./cert_for_test.pem"
				Settings.KeyPath = "./key_for_test.pem"
			}
			got, got1, err := GetOrCreateCertAndKey()
			if !tt.wantErr(t, err, fmt.Sprintf("GetOrCreateCertAndKey(%t)", tt.exist)) {
				return
			}
			assert.NotNil(t, got, "GetOrCreateCertAndKey()")
			assert.NotNil(t, got1, "GetOrCreateCertAndKey()")
		})
	}
}

func TestReadJSONConfig(t *testing.T) {
	type args struct {
		config   *Config
		filePath string
	}
	tests := []struct {
		wantErr assert.ErrorAssertionFunc
		want    *Config
		args    args
		name    string
		preload bool
	}{
		{
			name: "read json config success (no file exists)",
			args: args{
				config:   &Config{},
				filePath: "./config.json",
			},
			wantErr: assert.NoError,
			preload: false,
			want:    &Config{},
		},
		{
			name: "read json config success",
			args: args{
				config:   &Config{},
				filePath: "./config1.json",
			},
			wantErr: assert.NoError,
			preload: true,
			want: &Config{
				Address: "localhost:1337",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preload {
				file, err := os.OpenFile(tt.args.filePath, os.O_WRONLY|os.O_CREATE, 0600)
				defer func(file *os.File) {
					closeErr := file.Close()
					if closeErr != nil {
						assert.FailNow(t, "unexpected error closing file")
					}
				}(file)
				assert.NoError(t, err)
				_, writeErr := file.WriteString(`{"server_address": "localhost:1337"}`)
				assert.NoError(t, writeErr)
			}
			tt.wantErr(t, readJSONConfig(tt.args.config, tt.args.filePath), fmt.Sprintf("readJSONConfig(%v, %v)", tt.args.config, tt.args.filePath))
			assert.Equal(t, tt.want, tt.args.config)
		})
	}
}

func TestTLSEnabled_Set(t *testing.T) {
	tests := []struct {
		name string
		in   bool
	}{
		{
			name: "set tls enabled true success",
			in:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TLSEnabled{
				TLSEnabled: !tt.in,
			}
			err := e.Set(strconv.FormatBool(tt.in))
			assert.NoError(t, err)
			assert.Equal(t, e.TLSEnabled, tt.in)
		})
	}
}

func TestTLSEnabled_String(t *testing.T) {
	type fields struct {
		TLSEnabled bool
	}
	tests := []struct {
		name   string
		want   string
		fields fields
	}{
		{
			name: "get tls enabled success",
			fields: fields{
				TLSEnabled: true,
			},
			want: "true",
		},
		{
			name: "get tls enabled success",
			fields: fields{
				TLSEnabled: false,
			},
			want: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TLSEnabled{
				TLSEnabled: tt.fields.TLSEnabled,
			}
			assert.Equalf(t, tt.want, e.String(), "String()")
		})
	}
}

func TestTrustedSubnet_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "set trusted subnet success",
			input:   "192.168.0.0/24",
			wantErr: false,
		},
		{
			name:    "set trusted subnet empty",
			input:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &TrustedSubnet{}
			err := ts.Set(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.input, ts.CIDR)
		})
	}
}

func TestTrustedSubnet_String(t1 *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get trusted subnet success",
			want: "192.168.0.0/24",
		},
		{
			name: "get trusted subnet empty",
			want: "",
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TrustedSubnet{
				CIDR: tt.want,
			}
			assert.Equalf(t1, tt.want, t.String(), "String()")
		})
	}
}
