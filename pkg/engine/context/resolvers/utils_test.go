package resolvers

import (
	"reflect"
	"testing"
)

func TestGetCacheSelector(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{{
		name: "ok",
		want: LabelCacheKey,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCacheSelector()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCacheSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("GetCacheSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
