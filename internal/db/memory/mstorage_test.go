package memory

import (
	"errors"
	"testing"
)

func TestSet(t *testing.T) {
	type args[T any] struct {
		key  string
		val  *T
		m    *MStorage
		opts []func(*SetOptions)
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		wantErr error
	}
	type target struct {
		Key string
		Val int
	}
	ms := NewMemStorage()
	tests := []testCase[target]{
		{
			name: "default",
			args: args[target]{
				key:  "key1",
				val:  &target{Key: "key1", Val: 1},
				m:    ms,
				opts: nil,
			},
		}, {
			name: "duplicate records",
			args: args[target]{
				key:  "key1",
				val:  &target{Key: "key1", Val: 2},
				m:    ms,
				opts: nil,
			},
			wantErr: ErrDuplicateKey,
		}, {
			name: "overwrite",
			args: args[target]{
				key:  "key1",
				val:  &target{Key: "key1", Val: 3},
				m:    ms,
				opts: []func(*SetOptions){WithOverwrite()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Set[target](t.Context(), tt.args.key, tt.args.val, tt.args.m, tt.args.opts...)
			if err != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("%s: Set() error = %+v, wantErr %+v", tt.name, err, tt.wantErr)
			}

			if tt.wantErr == nil {
				val, getErr := Get[target](t.Context(), tt.args.key, tt.args.m)
				if getErr != nil {
					t.Fatal(getErr)
				}
				if val.Key != tt.args.val.Key || val.Val != tt.args.val.Val {
					t.Errorf("%s: Set() Val = %+v, want %+v", tt.name, val, tt.args.val)
				}
			}
		})
	}
}
