package user

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestCreateUserValidation(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name    string
		req     CreateUserRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CreateUserRequest{
				Name:     "Kamal Perera",
				Username: "kamalp",
				Password: "secretpassword",
				Role:     "advisor",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: CreateUserRequest{
				Username: "kamalp",
				Password: "secretpassword",
				Role:     "advisor",
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			req: CreateUserRequest{
				Name:     "Kamal Perera",
				Username: "kamalp",
				Password: "secretpassword",
				Role:     "admin", // not one of owner, advisor, mechanic
			},
			wantErr: true,
		},
		{
			name: "short password",
			req: CreateUserRequest{
				Name:     "Kamal Perera",
				Username: "kamalp",
				Password: "123",
				Role:     "mechanic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected wantErr=%v, got err=%v", tt.wantErr, err)
			}
		})
	}
}
