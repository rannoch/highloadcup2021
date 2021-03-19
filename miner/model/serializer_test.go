package model

import (
	"reflect"
	"testing"
)

func TestDig_MarshalJSON(t *testing.T) {
	type fields struct {
		LicenseID int32
		PosX      int32
		PosY      int32
		Depth     int32
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				LicenseID: 1,
				PosX:      2,
				PosY:      3,
				Depth:     4,
			},
			want:    []byte(`{"licenseID":1,"posX":2,"posY":3,"depth":4}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Dig{
				LicenseID: tt.fields.LicenseID,
				PosX:      tt.fields.PosX,
				PosY:      tt.fields.PosY,
				Depth:     tt.fields.Depth,
			}
			got, err := d.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}
