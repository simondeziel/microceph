package database

// The code below was generated by lxd-generate - DO NOT EDIT!

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/canonical/microcluster/cluster"
	"github.com/lxc/lxd/lxd/db/query"
	"github.com/lxc/lxd/shared/api"
)

var _ = api.ServerEnvironment{}

var diskObjects = cluster.RegisterStmt(`
SELECT disks.id, internal_cluster_members.name AS member, disks.osd, disks.path
  FROM disks
  JOIN internal_cluster_members ON disks.member_id = internal_cluster_members.id
  ORDER BY internal_cluster_members.id, disks.osd
`)

var diskObjectsByMember = cluster.RegisterStmt(`
SELECT disks.id, internal_cluster_members.name AS member, disks.osd, disks.path
  FROM disks
  JOIN internal_cluster_members ON disks.member_id = internal_cluster_members.id
  WHERE ( member = ? )
  ORDER BY internal_cluster_members.id, disks.osd
`)

var diskObjectsByMemberAndPath = cluster.RegisterStmt(`
SELECT disks.id, internal_cluster_members.name AS member, disks.osd, disks.path
  FROM disks
  JOIN internal_cluster_members ON disks.member_id = internal_cluster_members.id
  WHERE ( member = ? AND disks.path = ? )
  ORDER BY internal_cluster_members.id, disks.osd
`)

var diskID = cluster.RegisterStmt(`
SELECT disks.id FROM disks
  JOIN internal_cluster_members ON disks.member_id = internal_cluster_members.id
  WHERE internal_cluster_members.name = ? AND disks.osd = ?
`)

var diskCreate = cluster.RegisterStmt(`
INSERT INTO disks (member_id, osd, path)
  VALUES ((SELECT internal_cluster_members.id FROM internal_cluster_members WHERE internal_cluster_members.name = ?), ?, ?)
`)

var diskDeleteByMember = cluster.RegisterStmt(`
DELETE FROM disks WHERE member_id = (SELECT internal_cluster_members.id FROM internal_cluster_members WHERE internal_cluster_members.name = ?)
`)

var diskDeleteByMemberAndPath = cluster.RegisterStmt(`
DELETE FROM disks WHERE member_id = (SELECT internal_cluster_members.id FROM internal_cluster_members WHERE internal_cluster_members.name = ?) AND path = ?
`)

var diskUpdate = cluster.RegisterStmt(`
UPDATE disks
  SET member_id = (SELECT internal_cluster_members.id FROM internal_cluster_members WHERE internal_cluster_members.name = ?), osd = ?, path = ?
 WHERE id = ?
`)

// GetDisks returns all available disks.
// generator: disk GetMany
func GetDisks(ctx context.Context, tx *sql.Tx, filters ...DiskFilter) ([]Disk, error) {
	var err error

	// Result slice.
	objects := make([]Disk, 0)

	// Pick the prepared statement and arguments to use based on active criteria.
	var sqlStmt *sql.Stmt
	args := []any{}
	queryParts := [2]string{}

	if len(filters) == 0 {
		sqlStmt, err = cluster.Stmt(tx, diskObjects)
		if err != nil {
			return nil, fmt.Errorf("Failed to get \"diskObjects\" prepared statement: %w", err)
		}
	}

	for i, filter := range filters {
		if filter.Member != nil && filter.Path != nil && filter.OSD == nil {
			args = append(args, []any{filter.Member, filter.Path}...)
			if len(filters) == 1 {
				sqlStmt, err = cluster.Stmt(tx, diskObjectsByMemberAndPath)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"diskObjectsByMemberAndPath\" prepared statement: %w", err)
				}

				break
			}

			query, err := cluster.StmtString(diskObjectsByMemberAndPath)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"diskObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.Member != nil && filter.Path == nil && filter.OSD == nil {
			args = append(args, []any{filter.Member}...)
			if len(filters) == 1 {
				sqlStmt, err = cluster.Stmt(tx, diskObjectsByMember)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"diskObjectsByMember\" prepared statement: %w", err)
				}

				break
			}

			query, err := cluster.StmtString(diskObjectsByMember)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"diskObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.Member == nil && filter.Path == nil && filter.OSD == nil {
			return nil, fmt.Errorf("Cannot filter on empty DiskFilter")
		} else {
			return nil, fmt.Errorf("No statement exists for the given Filter")
		}
	}

	// Dest function for scanning a row.
	dest := func(scan func(dest ...any) error) error {
		d := Disk{}
		err := scan(&d.ID, &d.Member, &d.OSD, &d.Path)
		if err != nil {
			return err
		}

		objects = append(objects, d)

		return nil
	}

	// Select.
	if sqlStmt != nil {
		err = query.SelectObjects(ctx, sqlStmt, dest, args...)
	} else {
		queryStr := strings.Join(queryParts[:], "ORDER BY")
		err = query.Scan(ctx, tx, queryStr, dest, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"disks\" table: %w", err)
	}

	return objects, nil
}

// GetDisk returns the disk with the given key.
// generator: disk GetOne
func GetDisk(ctx context.Context, tx *sql.Tx, member string, osd int) (*Disk, error) {
	filter := DiskFilter{}
	filter.Member = &member
	filter.OSD = &osd

	objects, err := GetDisks(ctx, tx, filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"disks\" table: %w", err)
	}

	switch len(objects) {
	case 0:
		return nil, api.StatusErrorf(http.StatusNotFound, "Disk not found")
	case 1:
		return &objects[0], nil
	default:
		return nil, fmt.Errorf("More than one \"disks\" entry matches")
	}
}

// GetDiskID return the ID of the disk with the given key.
// generator: disk ID
func GetDiskID(ctx context.Context, tx *sql.Tx, member string, osd int) (int64, error) {
	stmt, err := cluster.Stmt(tx, diskID)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"diskID\" prepared statement: %w", err)
	}

	row := stmt.QueryRowContext(ctx, member, osd)
	var id int64
	err = row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return -1, api.StatusErrorf(http.StatusNotFound, "Disk not found")
	}

	if err != nil {
		return -1, fmt.Errorf("Failed to get \"disks\" ID: %w", err)
	}

	return id, nil
}

// DiskExists checks if a disk with the given key exists.
// generator: disk Exists
func DiskExists(ctx context.Context, tx *sql.Tx, member string, osd int) (bool, error) {
	_, err := GetDiskID(ctx, tx, member, osd)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// CreateDisk adds a new disk to the database.
// generator: disk Create
func CreateDisk(ctx context.Context, tx *sql.Tx, object Disk) (int64, error) {
	// Check if a disk with the same key exists.
	exists, err := DiskExists(ctx, tx, object.Member, object.OSD)
	if err != nil {
		return -1, fmt.Errorf("Failed to check for duplicates: %w", err)
	}

	if exists {
		return -1, api.StatusErrorf(http.StatusConflict, "This \"disks\" entry already exists")
	}

	args := make([]any, 3)

	// Populate the statement arguments.
	args[0] = object.Member
	args[1] = object.OSD
	args[2] = object.Path

	// Prepared statement to use.
	stmt, err := cluster.Stmt(tx, diskCreate)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"diskCreate\" prepared statement: %w", err)
	}

	// Execute the statement.
	result, err := stmt.Exec(args...)
	if err != nil {
		return -1, fmt.Errorf("Failed to create \"disks\" entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Failed to fetch \"disks\" entry ID: %w", err)
	}

	return id, nil
}

// DeleteDisk deletes the disk matching the given key parameters.
// generator: disk DeleteOne-by-Member-and-Path
func DeleteDisk(ctx context.Context, tx *sql.Tx, member string, path string) error {
	stmt, err := cluster.Stmt(tx, diskDeleteByMemberAndPath)
	if err != nil {
		return fmt.Errorf("Failed to get \"diskDeleteByMemberAndPath\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(member, path)
	if err != nil {
		return fmt.Errorf("Delete \"disks\": %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n == 0 {
		return api.StatusErrorf(http.StatusNotFound, "Disk not found")
	} else if n > 1 {
		return fmt.Errorf("Query deleted %d Disk rows instead of 1", n)
	}

	return nil
}

// DeleteDisks deletes the disk matching the given key parameters.
// generator: disk DeleteMany-by-Member
func DeleteDisks(ctx context.Context, tx *sql.Tx, member string) error {
	stmt, err := cluster.Stmt(tx, diskDeleteByMember)
	if err != nil {
		return fmt.Errorf("Failed to get \"diskDeleteByMember\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(member)
	if err != nil {
		return fmt.Errorf("Delete \"disks\": %w", err)
	}

	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	return nil
}

// UpdateDisk updates the disk matching the given key parameters.
// generator: disk Update
func UpdateDisk(ctx context.Context, tx *sql.Tx, member string, osd int, object Disk) error {
	id, err := GetDiskID(ctx, tx, member, osd)
	if err != nil {
		return err
	}

	stmt, err := cluster.Stmt(tx, diskUpdate)
	if err != nil {
		return fmt.Errorf("Failed to get \"diskUpdate\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(object.Member, object.OSD, object.Path, id)
	if err != nil {
		return fmt.Errorf("Update \"disks\" entry failed: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n != 1 {
		return fmt.Errorf("Query updated %d rows instead of 1", n)
	}

	return nil
}
