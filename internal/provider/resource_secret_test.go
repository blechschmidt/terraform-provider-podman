package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccCheckPodmanSecretSupported(t *testing.T) {
	t.Helper()
	config := testAccProviders["podman"].Meta()
	if config == nil {
		// Provider not yet configured; skip pre-flight and let the test itself fail
		// if secrets are truly unsupported.
		return
	}
	client := config.(*ProviderConfig).Client
	_, err := client.SecretList(context.Background(), types.SecretListOptions{})
	if err != nil && strings.Contains(err.Error(), "not supported") {
		t.Skip("Podman secrets not supported in this configuration")
	}
}

func testAccCheckPodmanSecretDestroy(s *terraform.State) error {
	config := testAccProviders["podman"].Meta()
	if config == nil {
		return nil
	}
	client := config.(*ProviderConfig).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_secret" {
			continue
		}

		_, _, err := client.SecretInspectWithRaw(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("secret %s still exists", rs.Primary.ID)
		}
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "not found") && !strings.Contains(errMsg, "no such") {
			return fmt.Errorf("unexpected error checking secret %s: %s", rs.Primary.ID, err)
		}
	}
	return nil
}

func TestAccPodmanSecret_basic(t *testing.T) {
	secretName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckPodmanSecretSupported(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_secret" "test" {
  name = "%s"
  data = "supersecret"
}
`, secretName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_secret.test", "id"),
					resource.TestCheckResourceAttr("podman_secret.test", "name", secretName),
				),
			},
		},
	})
}

func TestAccPodmanSecret_withLabels(t *testing.T) {
	secretName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckPodmanSecretSupported(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_secret" "test" {
  name = "%s"
  data = "supersecret"

  labels {
    label = "env"
    value = "testing"
  }

  labels {
    label = "app"
    value = "terraform"
  }
}
`, secretName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_secret.test", "id"),
					resource.TestCheckResourceAttr("podman_secret.test", "name", secretName),
					resource.TestCheckResourceAttr("podman_secret.test", "labels.#", "2"),
				),
			},
		},
	})
}
