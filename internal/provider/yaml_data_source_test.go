package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccYamlDataSourceSchemaDefaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Defaults are applied when enabled
			{
				Config: testAccYamlDataSourceSchemaDefaultsConfig(true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.terraconf_yaml.test",
						tfjsonpath.New("output"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.ObjectExact(map[string]knownvalue.Check{
								"name":            knownvalue.StringExact("shared"),
								"replicas":        knownvalue.Int64Exact(5),
								"inheritEntities": knownvalue.Bool(false),
							}),
						}),
					),
				},
			},
			// Defaults are not applied when disabled
			{
				Config: testAccYamlDataSourceSchemaDefaultsConfig(false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.terraconf_yaml.test",
						tfjsonpath.New("output"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.ObjectExact(map[string]knownvalue.Check{
								"name":     knownvalue.StringExact("shared"),
								"replicas": knownvalue.Int64Exact(5),
							}),
						}),
					),
				},
			},
		},
	})
}

func TestAccYamlDataSourceSchemaDefaultsRequiresSchema(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "terraconf_yaml" "test" {
  input_glob            = "./testdata/schema-defaults/data/*.yaml"
  apply_schema_defaults = true
}
`,
				ExpectError: regexp.MustCompile("Missing Schema File"),
			},
		},
	})
}

func testAccYamlDataSourceSchemaDefaultsConfig(applySchemaDefaults bool) string {
	if applySchemaDefaults {
		return `
data "terraconf_yaml" "test" {
  input_glob            = "./testdata/schema-defaults/data/*.yaml"
  schema_file           = "./testdata/schema-defaults/schema.json"
  apply_schema_defaults = true
  add_annotations       = false
}
`
	}

	return `
data "terraconf_yaml" "test" {
  input_glob      = "./testdata/schema-defaults/data/*.yaml"
  schema_file     = "./testdata/schema-defaults/schema.json"
  add_annotations = false
}
`
}
