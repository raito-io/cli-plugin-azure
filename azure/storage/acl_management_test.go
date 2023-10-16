package storage

import "testing"

func TestACLPermissionSet_String(t *testing.T) {
	tests := []struct {
		name string
		s    ACLPermissionSet
		want string
	}{
		{
			name: "No Permission",
			s:    ACLPermissionSet(0),
			want: "---",
		},
		{
			name: "Execute",
			s:    ACLPermissionSet(Execute),
			want: "--X",
		},
		{
			name: "Write",
			s:    ACLPermissionSet(Write),
			want: "-W-",
		},
		{
			name: "Read",
			s:    ACLPermissionSet(Read),
			want: "R--",
		},
		{
			name: "Execute and Write",
			s:    ACLPermissionSet(Execute).Add(Write),
			want: "-WX",
		},
		{
			name: "Execute and Read",
			s:    ACLPermissionSet(Execute).Add(Read),
			want: "R-X",
		},
		{
			name: "Write and Read",
			s:    ACLPermissionSet(Write).Add(Read),
			want: "RW-",
		},
		{
			name: "Execute and Write and Read",
			s:    ACLPermissionSet(Execute).Add(Write).Add(Read),
			want: "RWX",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
