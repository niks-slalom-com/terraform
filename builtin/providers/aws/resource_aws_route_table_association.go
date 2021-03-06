package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteTableAssociationCreate,
		Read:   resourceAwsRouteTableAssociationRead,
		Update: resourceAwsRouteTableAssociationUpdate,
		Delete: resourceAwsRouteTableAssociationDelete,

		Schema: map[string]*schema.Schema{
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	log.Printf(
		"[INFO] Creating route table association: %s => %s",
		d.Get("subnet_id").(string),
		d.Get("route_table_id").(string))

	resp, err := ec2conn.AssociateRouteTable(&ec2.AssociateRouteTableRequest{
		RouteTableID: aws.String(d.Get("route_table_id").(string)),
		SubnetID:     aws.String(d.Get("subnet_id").(string)),
	})

	if err != nil {
		return err
	}

	// Set the ID and return
	d.SetId(*resp.AssociationID)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	// Get the routing table that this association belongs to
	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(
		ec2conn, d.Get("route_table_id").(string))()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		return nil
	}
	rt := rtRaw.(*ec2.RouteTable)

	// Inspect that the association exists
	found := false
	for _, a := range rt.Associations {
		if *a.RouteTableAssociationID == d.Id() {
			found = true
			d.Set("subnet_id", *a.SubnetID)
			break
		}
	}

	if !found {
		// It seems it doesn't exist anymore, so clear the ID
		d.SetId("")
	}

	return nil
}

func resourceAwsRouteTableAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	log.Printf(
		"[INFO] Creating route table association: %s => %s",
		d.Get("subnet_id").(string),
		d.Get("route_table_id").(string))

	req := &ec2.ReplaceRouteTableAssociationRequest{
		AssociationID: aws.String(d.Id()),
		RouteTableID:  aws.String(d.Get("route_table_id").(string)),
	}
	resp, err := ec2conn.ReplaceRouteTableAssociation(req)

	if err != nil {
		ec2err, ok := err.(aws.APIError)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			// Not found, so just create a new one
			return resourceAwsRouteTableAssociationCreate(d, meta)
		}

		return err
	}

	// Update the ID
	d.SetId(*resp.NewAssociationID)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).awsEC2conn

	log.Printf("[INFO] Deleting route table association: %s", d.Id())
	err := ec2conn.DisassociateRouteTable(&ec2.DisassociateRouteTableRequest{
		AssociationID: aws.String(d.Id()),
	})
	if err != nil {
		ec2err, ok := err.(aws.APIError)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table association: %s", err)
	}

	return nil
}
