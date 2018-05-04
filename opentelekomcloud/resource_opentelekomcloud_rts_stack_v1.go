package opentelekomcloud

import (

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/huaweicloud/golangsdk/openstack/rts/v1/stacks"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/huaweicloud/golangsdk"
	"log"
	"fmt"
)

func resourceRtsStackV1() *schema.Resource {
	return &schema.Resource{
		Create:resourceRtsStackV1Create,
		Read:resourceRtsStackV1Read,
		Update:resourceRtsStackV1Update,
		Delete:resourceRtsStackV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{ //request and response parameters
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateName,
			},
			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed: 	true,
			},
			"template": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateStackTemplate,
				StateFunc: func(v interface{}) string {
					template, _ := normalizeStackTemplate(v)
					return template
				},
			},
			"environment": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
			},

			"parameters": &schema.Schema{
				Type:         schema.TypeMap,
				Optional:     true,
				Computed:		true,
			},
			"timeout_mins": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,

			},
			"disable_rollback": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"status_reason": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"outputs": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
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

		},
	}
}


func resourcetemplateV1(d *schema.ResourceData) *stacks.Template {
	rawTemplate := d.Get("template").(string)
	template := new(stacks.Template)
	template.Bin=[]byte(rawTemplate)
	log.Printf("[DEBUG] template: %s", template)
	return template
}

func resourceenvironmentV1(d *schema.ResourceData) *stacks.Environment {
	rawTemplate := d.Get("environment").(string)
	environment := new(stacks.Environment)
	environment.Bin=[]byte(rawTemplate)
	return environment
}
func resourceparameterV1(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("parameters").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}


func resourceRtsStackV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	orchestrationClient, err := config.orchestrationV1Client(GetRegion(d, config))

	log.Printf("[DEBUG] Value of orchestration client: %#v", orchestrationClient)

	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud orchestration client: %s", err)
	}

	rollback:=d.Get("disable_rollback").(bool)
	createOpts := stacks.CreateOpts{
		Name: 				d.Get("name").(string),
		TemplateOpts:		resourcetemplateV1(d),
		DisableRollback:	&rollback,
		EnvironmentOpts:	resourceenvironmentV1(d),
		Parameters:			resourceparameterV1(d),
		Timeout:			d.Get("timeout_mins").(int),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := stacks.Create(orchestrationClient, createOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud stack: %s", err)
	}

	log.Printf("[INFO] stack ID: %s", n.ID)

	log.Printf("[DEBUG] Waiting for OpenTelekomCloud stack (%s) to become available", n.ID)


	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATE_IN_PROGRESS",
							 "DELETE_IN_PROGRESS",
							 "ROLLBACK_IN_PROGRESS",},
		Target:     []string{"CREATE_COMPLETE",
							 "CREATE_FAILED",
							 "DELETE_COMPLETE",
							 "DELETE_FAILED",
							 "ROLLBACK_COMPLETE",
							 "ROLLBACK_FAILED",},
		Refresh:    waitForStackActive(orchestrationClient,d.Get("name").(string), n.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	d.SetId(n.ID)

	data, err1 := stacks.Get(orchestrationClient, d.Get("name").(string), n.ID).Extract()
	if err1!=nil {

	}
	if data.Status == "DELETE_COMPLETE" || data.Status == "DELETE_FAILED" {
		return fmt.Errorf("%s: %s", data.Status, data.StatusReason)
	}
	if data.Status == "CREATE_FAILED" || data.Status == "ROLLBACK_FAILED" {

		return fmt.Errorf("%s: %s", data.Status, data.StatusReason)
	}





	return resourceRtsStackV1Read(d, meta)

}

func resourceRtsStackV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	orchestrationClient, err := config.orchestrationV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud orchestration Client: %s", err)
	}

	n, err := stacks.Get(orchestrationClient, d.Get("name").(string),d.Id()).Extract()
	if err != nil {
		if _, ok := err.(golangsdk.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenTelekomCloud Stacks: %s", err)
	}
	if n.Status == "DELETE_COMPLETE" {
		return fmt.Errorf( "Removing  Stack %s" + " as it has been already deleted", d.Id())

	}
	log.Printf("[DEBUG] Retrieved Stack %s: %+v", d.Id(), n)

	d.Set("disable_rollback", n.DisableRollback)
	d.Set("description", n.Description)
	d.Set("parameters", n.Parameters)
	d.Set("status_reason", n.StatusReason)
	d.Set("name", n.Name)
	d.Set("outputs", n.Outputs)
	d.Set("capabilities", n.Capabilities)
	d.Set("notification_topics", n.NotificationTopics)
	d.Set("timeout_mins",n.Timeout)
	d.Set("status",n.Status)
	d.Set("id", n.ID)

	return nil
}

func resourceRtsStackV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	orchestrationClient, err := config.orchestrationV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud orchestration Client: %s", err)
	}

	var updateOpts stacks.UpdateOpts

	updateOpts.TemplateOpts = resourcetemplateV1(d)

	if d.HasChange("environment") {

		updateOpts.EnvironmentOpts = resourceenvironmentV1(d)
	}
	if d.HasChange("parameters") {

		updateOpts.Parameters =resourceparameterV1(d)
	}
	if d.HasChange("timeout_mins") {
		updateOpts.Timeout = d.Get("timeout_mins").(int)
	}
	if d.HasChange("disable_rollback") {
		rollback:=d.Get("disable_rollback").(bool)
		updateOpts.DisableRollback = &rollback
	}


	log.Printf("[DEBUG] Updating Stack %s with options: %+v", d.Id(), updateOpts)

		log.Printf("[DEBUG] Updating Stack %s with options: %#v", d.Id(), updateOpts)
		err = stacks.Update(orchestrationClient,d.Get("name").(string), d.Id(), updateOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error updating OpenTelekomCloud Stack: %s", err)
		}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"UPDATE_IN_PROGRESS",
							 "CREATE_COMPLETE",
							 "ROLLBACK_IN_PROGRESS"},
		Target:     []string{"UPDATE_COMPLETE",
							 "UPDATE_FAILED",
							 "ROLLBACK_COMPLETE",
							 "ROLLBACK_FAILED"},
		Refresh:    waitForStackUpdate(orchestrationClient,d.Get("name").(string), d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	data, err1 := stacks.Get(orchestrationClient, d.Get("name").(string), d.Id()).Extract()
	if err1!=nil {

	}
	if data.Status == "ROLLBACK_COMPLETE" || data.Status == "ROLLBACK_FAILED"|| data.Status == "UPDATE_FAILED" {

		return fmt.Errorf("%s: %s", data.Status, data.StatusReason)
	}

	return resourceRtsStackV1Read(d, meta)
}

func resourceRtsStackV1Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy Stack: %s", d.Id())

	config := meta.(*Config)
	orchestrationClient, err := config.orchestrationV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud orchestration Client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETE_IN_PROGRESS",
						     "CREATE_COMPLETE",
						     "CREATE_FAILED",
						     "UPDATE_COMPLETE",
						     "UPDATE_FAILED",
						     "CREATE_FAILED",
						     "ROLLBACK_COMPLETE",
						     "ROLLBACK_IN_PROGRESS"},
		Target:     []string{"DELETE_COMPLETE",
							 "DELETE_FAILED"},
		Refresh:    waitForStackDelete(orchestrationClient,d.Get("name").(string), d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenTelekomCloud Stack: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForStackActive(orchestrationClient *golangsdk.ServiceClient, stackName string,stackId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := stacks.Get(orchestrationClient, stackName,stackId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenTelekomCloud stack: %+v", n)
		if n.Status == "CREATE_IN_PROGRESS"  {
			return n,n.Status, nil
		}

		return n, n.Status, nil
	}
}

func waitForStackDelete(orchestrationClient *golangsdk.ServiceClient, stackName string,stackId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenTelekomCloud Stack %s.\n", stackId)

		r, err := stacks.Get(orchestrationClient , stackName , stackId).Extract()
		log.Printf("[DEBUG] Value after extract: %#v", r)
		if err != nil {
			if r.Status == "DELETE_COMPLETE" {
				log.Panic(" Removing  Stack %s" + " as it has been already deleted", stackId)
				return nil, "", err

			}
			return r, r.Status, err
		}
		err = stacks.Delete(orchestrationClient, stackName ,stackId).ExtractErr()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenTelekomCloud Stack %s", stackId)
				return r, r.Status, nil
			}
			if errCode, ok := err.(golangsdk.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return r, r.Status, nil
				}
			}
			return r, r.Status, err
		}

		log.Printf("[DEBUG] OpenTelekomCloud Stack %s still active.\n", stackId)
		return r, r.Status, nil
	}
}

func waitForStackUpdate(orchestrationClient *golangsdk.ServiceClient, stackName string,stackId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		n, err := stacks.Get(orchestrationClient, stackName,stackId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenTelekomCloud stack: %+v", n)
		if n.Status == "UPDATE_IN_PROGRESS"  {
			return n, "UPDATE_IN_PROGRESS", nil
		}

		return n, n.Status, nil
	}
}
