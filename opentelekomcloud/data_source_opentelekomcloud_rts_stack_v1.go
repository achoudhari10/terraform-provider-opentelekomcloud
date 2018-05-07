package opentelekomcloud
import (
	"fmt"
	"github.com/huaweicloud/golangsdk/openstack/rts/v1/stacks"
	"github.com/huaweicloud/golangsdk/openstack/rts/v1/stacktemplates"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"reflect"
	"unsafe"

	)

func dataSourceStackV1() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceStackV1Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status_reason": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"outputs": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"timeout_mins": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,

			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"disable_rollback": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"capabilities":  &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"notification_topics":  &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"template_body": {
				Type:     schema.TypeString,
				Computed: true,
			},

		},
	}
}


func dataSourceStackV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	orchestrationClient, err := config.orchestrationV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud rts client: %s", err)
	}
	listOpts := stacks.ListOpts{
		Status:        	d.Get("status").(string),
		Name: 			d.Get("name").(string),
	}
	pages, err := stacks.List(orchestrationClient, listOpts).AllPages()
	refinedStacks, err := stacks.ExtractStacks(pages)

	if err != nil {
		return fmt.Errorf("Unable to retrieve stacks: %s", err)
	}

	if len(refinedStacks) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(refinedStacks) > 1 {
		return fmt.Errorf("Your query returned more than one result." +
			" Please try a more specific search criteria")
	}
	stack := refinedStacks[0]
	log.Printf("[INFO] Retrieved Stacks using given filter %s: %+v", stack.ID, stack)
	d.SetId(stack.ID)

	d.Set("status", stack.Status)
	d.Set("name", stack.Name)
	d.Set("status_reason", stack.StatusReason)
	d.Set("description", stack.Description)
	d.Set("region", GetRegion(d, config))

	n, err := stacks.Get(orchestrationClient, stack.Name, stack.ID).Extract()
	log.Printf("[DEBUG] Retrieved n %+v", n)
	d.Set("disable_rollback", n.DisableRollback)
	d.Set("capabilities", n.Capabilities)
	d.Set("notification_topics", n.NotificationTopics)
	d.Set("timeout_mins", n.Timeout)


	TemplateList, err := stacktemplates.Get(orchestrationClient, stack.Name, stack.ID).Extract()
	log.Printf("[DEBUG] Retrieved TemplateList %+v", TemplateList)
	d.Set("outputs", flattenStackOutputs(n.Outputs))
	d.Set("parameters", n.Parameters)

	fmt.Print(string(TemplateList))
	template := BytesToString(TemplateList)
	log.Printf("value of str1", template)
	d.Set("template_body", template)
	log.Printf("template set data", d.Set("template_body", template))

	return nil
}

func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}