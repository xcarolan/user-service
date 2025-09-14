package models

import (
	"testing"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "valid user",
			user:    User{Name: "test", Email: "test@test.com"},
			wantErr: false,
		},
		{
			name:    "empty name",
			user:    User{Email: "test@test.com"},
			wantErr: true,
		},
		{
			name:    "empty email",
			user:    User{Name: "test"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			user:    User{Name: "test", Email: "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.user.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		name    string
		idStr   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid id",
			idStr:   "123",
			want:    123,
			wantErr: false,
		},
		{
			name:    "empty id",
			idStr:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid id",
			idStr:   "abc",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUserID(tt.idStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}
