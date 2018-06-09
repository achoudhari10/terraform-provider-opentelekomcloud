package opentelekomcloud

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/huaweicloud/golangsdk/openstack/sharedfilesystems/v2/shares"
)

// PASS
func TestAccOTCSfsV2_basic(t *testing.T) {
	var share shares.Share

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOTCSfsV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSfsV2_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOTCSfsV2Exists("opentelekomcloud_sfs_file_sharing_v2.sfs_1", &share),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "name", "sfs-test1"),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "share_proto", "NFS"),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "status", "available"),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "size", "1"),
				),
			},
		},
	})
}

func TestAccOTCSfsV2_update(t *testing.T) {
	var share shares.Share

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOTCSfsV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSfsV2_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOTCSfsV2Exists("opentelekomcloud_sfs_file_sharing_v2.sfs_1", &share),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "name", "sfs-test1"),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "size", "1"),
				),
			},
			resource.TestStep{
				Config: testAccSfsV2_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOTCSfsV2Exists("opentelekomcloud_sfs_file_sharing_v2.sfs_1", &share),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "name", "sfs-test2"),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_sfs_file_sharing_v2.sfs_1", "size", "2"),
				),
			},
		},
	})
}

// PASS
func TestAccOTCSfsV2_timeout(t *testing.T) {
	var share shares.Share

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOTCSfsV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSfsV2_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOTCSfsV2Exists("opentelekomcloud_sfs_file_sharing_v2.sfs_1", &share),
				),
			},
		},
	})
}

func testAccCheckOTCSfsV2Destroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	sfsClient, err := config.sfsV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud sfs client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opentelekomcloud_sfs_file_sharing_v2" {
			continue
		}

		_, err := shares.Get(sfsClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Share File still exists")
		}
	}

	return nil
}

func testAccCheckOTCSfsV2Exists(n string, share *shares.Share) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		sfsClient, err := config.sfsV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenTelekomCloud sfs client: %s", err)
		}

		found, err := shares.Get(sfsClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("share file not found")
		}

		*share = *found

		return nil
	}
}

var testAccSfsV2_basic = fmt.Sprintf(`
resource "opentelekomcloud_sfs_file_sharing_v2" "sfs_1" {
	share_proto = "NFS"
	size=1
	name="sfs-test1"
  	availability_zone="eu-de-01"
	access_to="%s" 
  	access_type="cert"
  	access_level="rw"
	description="sfs_c2c_test-file"
}
`,OS_VPC_ID)

var testAccSfsV2_update  = fmt.Sprintf(`
resource "opentelekomcloud_sfs_file_sharing_v2" "sfs_1" {
	share_proto = "NFS"
	size=2
	name="sfs-test2"
  	availability_zone="eu-de-01"
	access_to="%s" 
  	access_type="cert"
  	access_level="rw"
	description="sfs_c2c_test-file"
}
`,OS_VPC_ID)

var testAccSfsV2_timeout = fmt.Sprintf(`
resource "opentelekomcloud_sfs_file_sharing_v2" "sfs_1" {
	share_proto = "NFS"
	size=1
	name="sfs-test1"
  	availability_zone="eu-de-01"
	access_to="%s" 
  	access_type="cert"
  	access_level="rw"
	description="sfs_c2c_test-file"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}`,OS_VPC_ID)
