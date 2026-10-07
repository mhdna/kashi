package db

import "context"

// SaveRolePermissionsTx replaces a role's permission rows wholesale, so the
// role never briefly has a half-applied mix of old and new access.
func (store *SQLStore) SaveRolePermissionsTx(ctx context.Context, roleID int64, perms []RolePermission) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.DeleteRolePermissions(ctx, roleID); err != nil {
			return err
		}
		for _, p := range perms {
			err := q.UpsertRolePermission(ctx, UpsertRolePermissionParams{
				RoleID:  roleID,
				Section: p.Section,
				Page:    p.Page,
				CanView: p.CanView,
				CanEdit: p.CanEdit,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
