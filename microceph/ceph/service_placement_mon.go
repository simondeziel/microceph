package ceph

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microceph/microceph/common"
	"github.com/canonical/microceph/microceph/database"
)

type MonServicePlacement struct {
	Name string
}

// Populate json payload data to the service object.
func (msp *MonServicePlacement) PopulateParams(s common.StateInterface, payload string) error {
	return nil
}

// Check if host is hospitable to the new service to be enabled.
func (msp *MonServicePlacement) HospitalityCheck(s common.StateInterface) error {
	return genericHospitalityCheck(msp.Name)
}

// Initialise the new service.
func (msp *MonServicePlacement) ServiceInit(s common.StateInterface) error {
	return genericServiceInit(s, msp.Name)
}

// Perform Post Placement checks for the service
func (msp *MonServicePlacement) PostPlacementCheck(s common.StateInterface) error {
	return genericPostPlacementCheck(msp.Name)
}

// Perform DB updates to persist the service enablement changes.
func (msp *MonServicePlacement) DbUpdate(s common.StateInterface) error {
	// Update the database.
	err := s.ClusterState().Database.Transaction(s.ClusterState().Context, func(ctx context.Context, tx *sql.Tx) error {
		// Record the role.
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.ClusterState().Name(), Service: msp.Name})
		if err != nil {
			return fmt.Errorf("failed to record role: %w", err)
		}

		err = updateDbForMon(s, ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to record mon host: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
