package digitalocean

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanFloatingIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanFloatingIPCreate,
		Read:   resourceDigitalOceanFloatingIPRead,
		Update: resourceDigitalOceanFloatingIPUpdate,
		Delete: resourceDigitalOceanFloatingIPDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"droplet_id": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"ipv4_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDigitalOceanFloatingIPCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options

	opts := &godo.FloatingIPCreateRequest{
		Region:    d.Get("region").(string),
		DropletID: d.Get("droplet_id").(int),
	}

	log.Printf("[DEBUG] Floating IP create configuration: %#v", opts)
	floatingIP, _, err := client.FloatingIPs.Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating Floating IP: %s", err)
	}

	d.SetId(floatingIP.IP)
	log.Printf("[INFO] Floating IP address: %s", floatingIP.IP)

	return resourceDigitalOceanFloatingIPRead(d, meta)
}

func resourceDigitalOceanFloatingIPRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	floatingIP, _, err := client.FloatingIPs.Get(d.Id())
	if err != nil {
		// If the floatingIP is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Floating IP: %s", err)
	}

	d.Set("ipv4_address", floatingIP.IP)

	return nil
}

func resourceDigitalOceanFloatingIPUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	ip := d.Id()

	if d.HasChange("droplet_id") {
		newDropletID, oldDropletID := d.GetChange("droplet_id")

		var err error
		if newDropletID.(int) == 0 {
			_, _, err = client.FloatingIPActions.Unassign(ip)
		} else {
			_, _, err = client.FloatingIPActions.Assign(ip, newDropletID.(int))
		}
		if err != nil {
			return fmt.Errorf(
				"Error updating floating IP (%s) droplet assignment: %s", d.Id(), err)
		}
		_, err = WaitForFloatingIPAttribute(
			d,
			strconv.Itoa(newDropletID.(int)),
			[]string{strconv.Itoa(oldDropletID.(int))},
			"droplet_id",
			meta)
		if err != nil {
			return fmt.Errorf(
				"Error waiting for floating IP (%s) assignent to change: %s", d.Id(), err)
		}
	}
	return resourceDigitalOceanFloatingIPRead(d, meta)
}

func resourceDigitalOceanFloatingIPDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting Floating IP: %s", d.Id())
	_, err := client.FloatingIPs.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Floating IP: %s", err)
	}

	d.SetId("")
	return nil
}

func WaitForFloatingIPAttribute(
	d *schema.ResourceData,
	target string,
	pending []string,
	attribute string,
	meta interface{},
) (interface{}, error) {

	log.Printf(
		"[INFO] Waiting for floating IP (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    newFloatingIPStateRefreshFunc(d, attribute, meta),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,

		// TODO(aybabtme): not sure this is required, investigate:
		//   https://github.com/hashicorp/terraform/issues/481
		NotFoundChecks: 60,
	}

	return stateConf.WaitForState()
}

func newFloatingIPStateRefreshFunc(
	d *schema.ResourceData,
	attribute string,
	meta interface{},
) resource.StateRefreshFunc {

	client := meta.(*godo.Client)
	ip := d.Id()

	return func() (interface{}, string, error) {

		err := resourceDigitalOceanFloatingIPRead(d, meta)
		if err != nil {
			return nil, "", err
		}

		// See if we can access our attribute
		if attr, ok := d.GetOk(attribute); ok {
			// Retrieve the floating IP properties
			floatingIP, _, err := client.FloatingIPs.Get(ip)
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving floating IP: %s", err)
			}
			switch typedAttr := attr.(type) {
			case string:
				return &floatingIP, typedAttr, nil
			case int:
				return &floatingIP, strconv.Itoa(typedAttr), nil
			default:
				return nil, "", fmt.Errorf("Error reading attribute %s for floating IP, unexpected type %T", attribute, attr)
			}
		}

		return nil, "", nil
	}
}
