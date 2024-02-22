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
			want: "--x",
		},
		{
			name: "Write",
			s:    ACLPermissionSet(Write),
			want: "-w-",
		},
		{
			name: "Read",
			s:    ACLPermissionSet(Read),
			want: "r--",
		},
		{
			name: "Execute and Write",
			s:    ACLPermissionSet(Execute).Add(Write),
			want: "-wx",
		},
		{
			name: "Execute and Read",
			s:    ACLPermissionSet(Execute).Add(Read),
			want: "r-x",
		},
		{
			name: "Write and Read",
			s:    ACLPermissionSet(Write).Add(Read),
			want: "rw-",
		},
		{
			name: "Execute and Write and Read",
			s:    ACLPermissionSet(Execute).Add(Write).Add(Read),
			want: "rwx",
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
