package schema

import (
	"testing"

	"entgo.io/ent"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPlatformWriteValidationSchemas(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fields  []ent.Field
		allowed func(string) bool
	}{
		{"account", Account{}.Fields(), domain.IsConcretePlatform},
		{"group", Group{}.Fields(), domain.IsGroupPlatform},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, f := range tc.fields {
				d := f.Descriptor()
				if d.Name != "platform" {
					continue
				}
				for _, platform := range append(domain.ConcretePlatformIDs(), "composite", "bogus", "", "openai ") {
					var err error
					for _, validate := range d.Validators {
						validator, ok := validate.(func(string) error)
						require.True(t, ok, "unexpected platform validator type")
						err = validator(platform)
						if err != nil {
							break
						}
					}
					if tc.allowed(platform) {
						require.NoError(t, err, platform)
					} else {
						require.Error(t, err, platform)
					}
				}
				return
			}
			t.Fatal("missing platform field")
		})
	}
}
