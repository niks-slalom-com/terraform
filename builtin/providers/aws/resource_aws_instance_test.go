package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSInstance_normal(t *testing.T) {
	var v ec2.Instance

	testCheck := func(*terraform.State) error {
		if *v.Placement.AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("bad availability zone: %#v", *v.Placement.AvailabilityZone)
		}

		if len(v.SecurityGroups) == 0 {
			return fmt.Errorf("no security groups: %#v", v.SecurityGroups)
		}
		if *v.SecurityGroups[0].GroupName != "tf_test_foo" {
			return fmt.Errorf("no security groups: %#v", v.SecurityGroups)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					testCheck,
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"user_data",
						"0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"),
				),
			},

			// We repeat the exact same test so that we can be sure
			// that the user data hash stuff is working without generating
			// an incorrect diff.
			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					testCheck,
					resource.TestCheckResourceAttr(
						"aws_instance.foo",
						"user_data",
						"0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"),
				),
			},
		},
	})
}

func TestAccAWSInstance_blockDevices(t *testing.T) {
	var v ec2.Instance

	testCheck := func() resource.TestCheckFunc {
		return func(*terraform.State) error {

			// Map out the block devices by name, which should be unique.
			blockDevices := make(map[string]ec2.InstanceBlockDeviceMapping)
			for _, blockDevice := range v.BlockDeviceMappings {
				blockDevices[*blockDevice.DeviceName] = blockDevice
			}

			// Check if the root block device exists.
			if _, ok := blockDevices["/dev/sda1"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sda1")
			}

			// Check if the secondary block device exists.
			if _, ok := blockDevices["/dev/sdb"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sdb")
			}

			// Check if the third block device exists.
			if _, ok := blockDevices["/dev/sdc"]; !ok {
				fmt.Errorf("block device doesn't exist: /dev/sdc")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigBlockDevices,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.device_name", "/dev/sda1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_size", "11"),
					// this one is important because it's the only root_block_device
					// attribute that comes back from the API. so checking it verifies
					// that we set state properly
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "root_block_device.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.172787947.device_name", "/dev/sdb"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.172787947.volume_size", "9"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.172787947.iops", "0"),
					// Check provisioned SSD device
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.3336996981.volume_type", "io1"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.3336996981.device_name", "/dev/sdc"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.3336996981.volume_size", "10"),
					resource.TestCheckResourceAttr(
						"aws_instance.foo", "block_device.3336996981.iops", "100"),
					testCheck(),
				),
			},
		},
	})
}

func TestAccAWSInstance_sourceDestCheck(t *testing.T) {
	var v ec2.Instance

	testCheck := func(enabled bool) resource.TestCheckFunc {
		return func(*terraform.State) error {
			if *v.SourceDestCheck != enabled {
				return fmt.Errorf("bad source_dest_check: %#v", *v.SourceDestCheck)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigSourceDestDisable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(false),
				),
			},

			resource.TestStep{
				Config: testAccInstanceConfigSourceDestEnable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(true),
				),
			},

			resource.TestStep{
				Config: testAccInstanceConfigSourceDestDisable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheck(false),
				),
			},
		},
	})
}

func TestAccAWSInstance_vpc(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
				),
			},
		},
	})
}

func TestAccInstance_NetworkInstanceSecurityGroups(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceNetworkInstanceSecurityGroups,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo_instance", &v),
				),
			},
		},
	})
}

func TestAccAWSInstance_tags(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckInstanceConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testAccCheckTagsSDK(&v.Tags, "foo", "bar"),
					// Guard against regression of https://github.com/hashicorp/terraform/issues/914
					testAccCheckTagsSDK(&v.Tags, "#", ""),
				),
			},

			resource.TestStep{
				Config: testAccCheckInstanceConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testAccCheckTagsSDK(&v.Tags, "foo", ""),
					testAccCheckTagsSDK(&v.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func TestAccAWSInstance_privateIP(t *testing.T) {
	var v ec2.Instance

	testCheckPrivateIP := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if *v.PrivateIPAddress != "10.1.1.42" {
				return fmt.Errorf("bad private IP: %s", *v.PrivateIPAddress)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigPrivateIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckPrivateIP(),
				),
			},
		},
	})
}

func TestAccAWSInstance_associatePublicIPAndPrivateIP(t *testing.T) {
	var v ec2.Instance

	testCheckPrivateIP := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			if *v.PrivateIPAddress != "10.1.1.42" {
				return fmt.Errorf("bad private IP: %s", *v.PrivateIPAddress)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigAssociatePublicIPAndPrivateIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists("aws_instance.foo", &v),
					testCheckPrivateIP(),
				),
			},
		},
	})
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).awsEC2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_instance" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesRequest{
			InstanceIDs: []string{rs.Primary.ID},
		})
		if err == nil {
			if len(resp.Reservations) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(aws.APIError)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidInstanceID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckInstanceExists(n string, i *ec2.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).awsEC2conn
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesRequest{
			InstanceIDs: []string{rs.Primary.ID},
		})
		if err != nil {
			return err
		}
		if len(resp.Reservations) == 0 {
			return fmt.Errorf("Instance not found")
		}

		*i = resp.Reservations[0].Instances[0]

		return nil
	}
}

func TestInstanceTenancySchema(t *testing.T) {
	actualSchema := resourceAwsInstance().Schema["tenancy"]
	expectedSchema := &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Computed: true,
		ForceNew: true,
	}
	if !reflect.DeepEqual(actualSchema, expectedSchema) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			actualSchema,
			expectedSchema)
	}
}

const testAccInstanceConfig = `
resource "aws_security_group" "tf_test_foo" {
	name = "tf_test_foo"
	description = "foo"

	ingress {
		protocol = "icmp"
		from_port = -1
		to_port = -1
		cidr_blocks = ["0.0.0.0/0"]
	}
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	availability_zone = "us-west-2a"

	instance_type = "m1.small"
	security_groups = ["${aws_security_group.tf_test_foo.name}"]
	user_data = "foo:-with-character's"
}
`

const testAccInstanceConfigBlockDevices = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-55a7ea65"
	instance_type = "m1.small"
	root_block_device {
		device_name = "/dev/sda1"
		volume_type = "gp2"
		volume_size = 11
	}
	block_device {
		device_name = "/dev/sdb"
		volume_size = 9
	}
	block_device {
		device_name = "/dev/sdc"
		volume_size = 10
		volume_type = "io1"
		iops = 100
	}
}
`

const testAccInstanceConfigSourceDestEnable = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
	source_dest_check = true
}
`

const testAccInstanceConfigSourceDestDisable = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
	source_dest_check = false
}
`

const testAccInstanceConfigVPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
	associate_public_ip_address = true
	tenancy = "dedicated"
}
`

const testAccCheckInstanceConfigTags = `
resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	tags {
		foo = "bar"
	}
}
`

const testAccCheckInstanceConfigTagsUpdate = `
resource "aws_instance" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	tags {
		bar = "baz"
	}
}
`

const testAccInstanceConfigPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
	private_ip = "10.1.1.42"
}
`

const testAccInstanceConfigAssociatePublicIPAndPrivateIP = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	ami = "ami-c5eabbf5"
	instance_type = "t2.micro"
	subnet_id = "${aws_subnet.foo.id}"
	associate_public_ip_address = true
	private_ip = "10.1.1.42"
}
`

const testAccInstanceNetworkInstanceSecurityGroups = `
resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-network-test"
	}
}

resource "aws_security_group" "tf_test_foo" {
  name = "tf_test_foo"
  description = "foo"
  vpc_id="${aws_vpc.foo.id}"

  ingress {
    protocol = "icmp"
    from_port = -1
    to_port = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_subnet" "foo" {
  cidr_block = "10.1.1.0/24"
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo_instance" {
  ami = "ami-21f78e11"
  instance_type = "t1.micro"
  security_groups = ["${aws_security_group.tf_test_foo.id}"]
  subnet_id = "${aws_subnet.foo.id}"
  associate_public_ip_address = true
	depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_eip" "foo_eip" {
  instance = "${aws_instance.foo_instance.id}"
  vpc = true
	depends_on = ["aws_internet_gateway.gw"]
}
`
