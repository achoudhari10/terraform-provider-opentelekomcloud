package opentelekomcloud
import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// PASS
func TestAccOpenTelekomCloudRtsStackV1DataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccOpenTelekomCloudRtsStackV1DataSource_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtsStackV1DataSourceID("data.opentelekomcloud_rts_stack_v1.stacks"),
					resource.TestCheckResourceAttr("data.opentelekomcloud_rts_stack_v1.stacks", "name", "opentelekomcloud_rts_stacktest"),
					resource.TestCheckResourceAttr("data.opentelekomcloud_rts_stack_v1.stacks", "description", "A HOT template that create a single server and boot from volume."),
					resource.TestCheckResourceAttr("data.opentelekomcloud_rts_stack_v1.stacks", "disable_rollback", "true"),
					resource.TestCheckResourceAttr("data.opentelekomcloud_rts_stack_v1.stacks", "parameters.%", "4"),
				),
			},
		},
	})
}

func testAccCheckRtsStackV1DataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find rts data source: %s ", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Rts data source ID not set ")
		}

		return nil
	}
}

var testAccOpenTelekomCloudRtsStackV1DataSource_basic = `
resource "opentelekomcloud_rts_stack_v1" "stack_1" {
  name = "opentelekomcloud_rts_stacktest"
  disable_rollback= true
  timeout_mins=60
  template = <<JSON
          {
			"outputs": {
              "str1": {
                 "description": "The description of the nat server.",
                 "value": {
                   "get_resource": "random"
                 }
	          }
            },
            "heat_template_version": "2013-05-23",
            "description": "A HOT template that create a single server and boot from volume.",
            "parameters": {
              "key_name": {
                "type": "string",
                "description": "Name of existing key pair for the instance to be created.",
                "default": "KeyPair-click2cloud"
	          }
	        },
            "resources": {
               "random": {
                  "type": "OS::Heat::RandomString",
                  "properties": {
                  "length": "6"
                  }
	          }
	       }
}
JSON
}

data "opentelekomcloud_rts_stack_v1" "stacks" {

        name = "${opentelekomcloud_rts_stack_v1.stack_1.name}"
       
}
`


