package digitalocean

import (
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanFloatingIP_Basic(t *testing.T) {
	var floatingIP godo.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanFloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDigitalOceanFloatingIPConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDigitalOceanFloatingIPExists("digitalocean_floating_ip.foobar", &floatingIP),
					testAccCheckDigitalOceanFloatingIPAttributes(&floatingIP),
					resource.TestCheckResourceAttr(
						"digitalocean_floating_ip.foobar", "name", "foobar-test-terraform.com"),
					resource.TestCheckResourceAttr(
						"digitalocean_floating_ip.foobar", "ip_address", "192.168.0.10"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanFloatingIPDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_floating_ip" {
			continue
		}

		// Try to find the floating ip
		_, _, err := client.FloatingIPs.Get(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("FloatingIP still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanFloatingIPAttributes(floatingIP *godo.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if floatingIP.Name != "foobar-test-terraform.com" {
			return fmt.Errorf("Bad name: %s", floatingIP.Name)
		}

		return nil
	}
}

func testAccCheckDigitalOceanFloatingIPExists(n string, floatingIP *godo.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		foundFloatingIP, _, err := client.FloatingIPs.Get(rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundFloatingIP.Name != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*floating_ip = *foundFloatingIP

		return nil
	}
}

const testAccCheckDigitalOceanFloatingIPConfig_basic = `
resource "digitalocean_floating_ip" "foobar" {
    name = "foobar-test-terraform.com"
    ip_address = "192.168.0.10"
}`
