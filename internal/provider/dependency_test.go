package provider

// This file holds shared helpers for the dependency integration tests
// (dependency_*_test.go). These tests exercise complex dependencies between
// resources and assert that changing an upstream resource correctly forces the
// dependent resource to be recreated (ForceNew) or updated in place.

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// depImage is a small, fast image used as the workhorse for container tests.
// It is pre-pulled by the acceptance workflow and is tiny, so create/recreate
// cycles stay quick.
const depImage = "docker.io/library/alpine:3.20"

// depImageAlt is a second tag of the same repository, used for image-change
// tests. "docker.io/library/alpine" (latest) is a prefix of depImage, which is
// exactly the shape that regressed container recreation.
const depImageLatest = "docker.io/library/alpine"

// randDepName returns a unique resource name so dependency tests can run in
// parallel without colliding on container/network/volume names.
func randDepName(prefix string) string {
	return fmt.Sprintf("tf-dep-%s-%s", prefix, acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
}

// testAccDepClient builds a docker/podman API client from the environment.
func testAccDepClient(t *testing.T) *client.Client {
	t.Helper()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("failed to create podman API client: %s", err)
	}
	return cli
}

// testAccDepPullImages pre-pulls the given image refs so that a podman_image
// resource whose name changes to one of them can Read it locally (the image
// resource only re-pulls when triggers/build change, not on a name change).
func testAccDepPullImages(t *testing.T, refs ...string) {
	t.Helper()
	cli := testAccDepClient(t)
	defer cli.Close()
	for _, ref := range refs {
		r, err := cli.ImagePull(context.Background(), ref, image.PullOptions{})
		if err != nil {
			t.Fatalf("failed to pull %s: %s", ref, err)
		}
		_, _ = io.Copy(io.Discard, r)
		_ = r.Close()
	}
}

// captureID stores the primary ID of the named resource into dst when the test
// step's check runs. Used to detect recreation across steps.
func captureID(resourceName string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID", resourceName)
		}
		*dst = rs.Primary.ID
		return nil
	}
}

// requireRecreated asserts that the resource's ID changed between two captures,
// i.e. a ForceNew attribute change caused the resource to be destroyed and
// recreated. label is used only for clearer failure messages.
func requireRecreated(label string, before, after *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *before == "" || *after == "" {
			return fmt.Errorf("%s: missing captured IDs (before=%q after=%q)", label, *before, *after)
		}
		if *before == *after {
			return fmt.Errorf("%s: expected resource to be recreated, but ID stayed %s", label, *before)
		}
		return nil
	}
}

// requireNotRecreated asserts that the resource's ID is unchanged between two
// captures, i.e. the change was applied in place rather than via replacement.
func requireNotRecreated(label string, before, after *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *before == "" || *after == "" {
			return fmt.Errorf("%s: missing captured IDs (before=%q after=%q)", label, *before, *after)
		}
		if *before != *after {
			return fmt.Errorf("%s: expected in-place update, but resource was recreated (%s -> %s)", label, *before, *after)
		}
		return nil
	}
}

// depRecreateTest describes the common two-step dependency test: apply configA,
// change something, apply configB, and assert the tracked resource was either
// recreated (ForceNew replacement) or updated in place.
type depRecreateTest struct {
	// resourceName is the address of the resource whose recreation is asserted,
	// e.g. "podman_container.test".
	resourceName string
	// label is used in failure messages.
	label string
	// configA and configB are the HCL for step 1 and step 2.
	configA string
	configB string
	// recreate is true when configB is expected to replace the resource, false
	// when it should be updated in place (ID stable).
	recreate bool
	// pulls lists image refs to pre-pull (needed when an image resource's name
	// changes to a tag that must already exist locally).
	pulls []string
}

// runDepRecreateTest runs a depRecreateTest against the live podman socket.
func runDepRecreateTest(t *testing.T, tc depRecreateTest) {
	t.Helper()
	var before, after string

	step2 := []resource.TestCheckFunc{captureID(tc.resourceName, &after)}
	if tc.recreate {
		step2 = append(step2, requireRecreated(tc.label, &before, &after))
	} else {
		step2 = append(step2, requireNotRecreated(tc.label, &before, &after))
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if len(tc.pulls) > 0 {
				testAccDepPullImages(t, tc.pulls...)
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: tc.configA,
				Check:  captureID(tc.resourceName, &before),
			},
			{
				Config: tc.configB,
				Check:  resource.ComposeTestCheckFunc(step2...),
			},
		},
	})
}

// testAccCheckPodmanContainerExists verifies that the named container resource
// exists in podman.
func testAccCheckPodmanContainerExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return err
		}
		defer cli.Close()
		if _, err := cli.ContainerInspect(context.Background(), rs.Primary.ID); err != nil {
			return fmt.Errorf("container %s (%s) not found: %w", resourceName, rs.Primary.ID, err)
		}
		return nil
	}
}
