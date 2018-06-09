package opentelekomcloud

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/sharedfilesystems/v2/shares"
	"log"
	"time"
)

func resourceSFSFileSharingV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceSFSSharingV2Create,
		Read:   resourceSFSSharingV2Read,
		Update: resourceSFSSharingV2Update,
		Delete: resourceSFSSharingV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"share_proto": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:"NFS",
			},
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_public": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"share_network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"consistency_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"access_level": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"access_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "cert",
			},
			"access_to": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"access_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"access_state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"export_location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"export_locations": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
			},
		},
	}
}

func resourceSFSSharingV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sfsV2Client(GetRegion(d, config))

	log.Printf("[DEBUG] Value of SFS Client: %#v", sfsClient)

	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud File Share Client: %s", err)
	}

	createOpts := shares.CreateOpts{
		ShareProto:         d.Get("share_proto").(string),
		Size:               d.Get("size").(int),
		Name:               d.Get("name").(string),
		Description:        d.Get("description").(string),
		SnapshotID:         d.Get("snapshot_id").(string),
		IsPublic:           d.Get("is_public").(bool),
		Metadata:           resourceSFSMetadataV2(d),
		ShareNetworkID:     d.Get("share_network_id").(string),
		ConsistencyGroupID: d.Get("consistency_group_id").(string),
		AvailabilityZone:   d.Get("availability_zone").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	create, err := shares.Create(sfsClient, createOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud File Share: %s", err)
	}
	d.SetId(create.ID)
	log.Printf("[INFO] Share ID: %s", create.Name)

	log.Printf("[DEBUG] Waiting for OpenTelekomCloud SFS File Share (%s) to become available", create.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Creating"},
		Target:     []string{"Available"},
		Refresh:    waitForSFSFileActive(sfsClient, create.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()

	grantAccessOpts := shares.GrantAccessOpts{
		AccessLevel: d.Get("access_level").(string),
		AccessType:  d.Get("access_type").(string),
		AccessTo:    d.Get("access_to").(string),
	}

	log.Printf("[DEBUG] Grant Access Rules: %#v", grantAccessOpts)
	grant, accessErr := shares.GrantAccess(sfsClient, d.Id(), grantAccessOpts).Extract()

	if accessErr != nil {
		return fmt.Errorf("Error applying access rules to share file : %s", accessErr)
	}

	log.Printf("[DEBUG] Waiting for OpenTelekomCloud SFS File Share (%s) to become available", grant.ID)

	return resourceSFSSharingV2Read(d, meta)

}

func resourceSFSSharingV2Read(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	sfsClient, err := config.sfsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud Vpc client: %s", err)
	}

	n, err := shares.Get(sfsClient, d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenTelekomCloud Shares: %s", err)
	}

	rules, err := shares.ListAccessRights(sfsClient, d.Id()).ExtractAccessRights()

	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenTelekomCloud Shares: %s", err)
	}

	//expLocs := n.ExportLocations
	//ExpLocations := make([]string, len(expLocs))
	//for i, val := range expLocs {
	//	ExpLocations[i] = val
	//}

	d.Set("id", n.ID)
	d.Set("name", n.Name)
	d.Set("share_proto", n.ShareProto)
	d.Set("status", n.Status)
	d.Set("size", n.Size)
	d.Set("description", n.Description)
	d.Set("share_type", n.ShareType)
	d.Set("volume_type", n.VolumeType)
	d.Set("snapshot_id", n.SnapshotID)
	d.Set("is_public", n.IsPublic)
	d.Set("metadata", n.Metadata)
	d.Set("share_network_id", n.ShareNetworkID)
	d.Set("availability_zone", n.AvailabilityZone)
	d.Set("region", GetRegion(d, config))
	d.Set("export_location", n.ExportLocation)
	d.Set("export_locations", n.ExportLocations)
	d.Set("host", n.Host)
	d.Set("links", n.Links)

	if len(rules) > 0 {
		rule := rules[0]
		d.Set("access_id", rule.ID)
		d.Set("access_state", rule.State)
	}
	return nil
}

func resourceSFSSharingV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sfsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error updating OpenTelekomCloud Share File: %s", err)
	}
	var updateOpts shares.UpdateOpts

	updateOpts.DisplayName = d.Get("name").(string)

	if d.HasChange("description") {
		updateOpts.DisplayDescription = d.Get("description").(string)
	}
	if d.HasChange("access_to") {
		deleteAccessOpts := shares.DeleteAccessOpts{AccessID: d.Get("access_id").(string)}
		deny := shares.DeleteAccess(sfsClient, d.Id(), deleteAccessOpts)
		if deny.Err != nil {
			return fmt.Errorf("Error changing access rules for share file : %s", deny.Err)
		}

		grantAccessOpts := shares.GrantAccessOpts{
			AccessLevel: d.Get("access_level").(string),
			AccessType:  d.Get("access_type").(string),
			AccessTo:    d.Get("access_to").(string),
		}

		log.Printf("[DEBUG] Grant Access Rules: %#v", grantAccessOpts)
		_, accessErr := shares.GrantAccess(sfsClient, d.Id(), grantAccessOpts).Extract()

		if accessErr != nil {
			return fmt.Errorf("Error changing access rules for share file : %s", accessErr)
		}
	}

	if d.HasChange("size") {
		old, new := d.GetChange("size")
		if old.(int) < new.(int) {
			expandOpts := shares.ExpandOpts{OSExtend: shares.OSExtendOpts{NewSize: new.(int)}}
			expand := shares.Expand(sfsClient, d.Id(), expandOpts)
			if expand.Err != nil {
				return fmt.Errorf("Error Expanding OpenTelekomCloud Share File size: %s", expand.Err)
			}
		} else {
			shrinkOpts := shares.ShrinkOpts{OSShrink: shares.OSShrinkOpts{NewSize: new.(int)}}
			shrink := shares.Shrink(sfsClient, d.Id(), shrinkOpts)
			if shrink.Err != nil {
				return fmt.Errorf("Error Shrinking OpenTelekomCloud Share File size: %s", shrink.Err)
			}
		}
	}

	_, err = shares.Update(sfsClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenTelekomCloud Share File: %s", err)
	}
	return resourceSFSSharingV2Read(d, meta)
}

func resourceSFSSharingV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sfsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud Shared File: %s", err)
	}
	share_id := d.Get("id").(string)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSFSFileDelete(sfsClient, share_id),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenTelekomCloud Share File: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForSFSFileActive(sfsClient *golangsdk.ServiceClient, shareID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := shares.Get(sfsClient, shareID).Extract()
		if err != nil {
			return nil, "", err
		}

		if n.Status == "OK" {
			return n, "ACTIVE", nil
		}

		if n.Status == "DOWN" {
			return nil, "", fmt.Errorf("Share File status: '%s'", n.Status)
		}

		return n, n.Status, nil
	}
}

func waitForSFSFileDelete(sfsClient *golangsdk.ServiceClient, shareId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		r, err := shares.Get(sfsClient, shareId).Extract()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted OpenTelekomCloud shared File %s", shareId)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}
		err = shares.Delete(sfsClient, shareId).ExtractErr()

		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[INFO] Successfully deleted OpenTelekomCloud shared File %s", shareId)
				return r, "DELETED", nil
			}
			if errCode, ok := err.(golangsdk.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return r, "ACTIVE", nil
				}
			}
			return r, "ACTIVE", err
		}

		return r, "ACTIVE", nil
	}
}

func resourceSFSMetadataV2(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}
