package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *Driver) CreateResource(ctx context.Context, create *store.Resource) (*store.Resource, error) {
	stmt := `
		INSERT INTO resource (
			filename,
			blob,
			external_link,
			type,
			size,
			creator_id,
			internal_path
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_ts, updated_ts
	`
	if err := d.db.QueryRowContext(
		ctx,
		stmt,
		create.Filename,
		create.Blob,
		create.ExternalLink,
		create.Type,
		create.Size,
		create.CreatorID,
		create.InternalPath,
	).Scan(&create.ID, &create.CreatedTs, &create.UpdatedTs); err != nil {
		return nil, err
	}

	return create, nil
}

func (d *Driver) ListResources(ctx context.Context, find *store.FindResource) ([]*store.Resource, error) {
	where, args := []string{"1 = 1"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "creator_id = ?"), append(args, *v)
	}
	if v := find.Filename; v != nil {
		where, args = append(where, "filename = ?"), append(args, *v)
	}
	if v := find.MemoID; v != nil {
		where, args = append(where, "memo_id = ?"), append(args, *v)
	}
	if find.HasRelatedMemo {
		where = append(where, "memo_id IS NOT NULL")
	}

	fields := []string{"id", "filename", "external_link", "type", "size", "creator_id", "created_ts", "updated_ts", "internal_path", "memo_id"}
	if find.GetBlob {
		fields = append(fields, "blob")
	}

	query := fmt.Sprintf(`
		SELECT
			%s
		FROM resource
		WHERE %s
		GROUP BY id
		ORDER BY created_ts DESC
	`, strings.Join(fields, ", "), strings.Join(where, " AND "))
	if find.Limit != nil {
		query = fmt.Sprintf("%s LIMIT %d", query, *find.Limit)
		if find.Offset != nil {
			query = fmt.Sprintf("%s OFFSET %d", query, *find.Offset)
		}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*store.Resource, 0)
	for rows.Next() {
		resource := store.Resource{}
		var memoID sql.NullInt32
		dests := []any{
			&resource.ID,
			&resource.Filename,
			&resource.ExternalLink,
			&resource.Type,
			&resource.Size,
			&resource.CreatorID,
			&resource.CreatedTs,
			&resource.UpdatedTs,
			&resource.InternalPath,
			&memoID,
		}
		if find.GetBlob {
			dests = append(dests, &resource.Blob)
		}
		if err := rows.Scan(dests...); err != nil {
			return nil, err
		}
		if memoID.Valid {
			resource.MemoID = &memoID.Int32
		}
		list = append(list, &resource)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *Driver) UpdateResource(ctx context.Context, update *store.UpdateResource) (*store.Resource, error) {
	set, args := []string{}, []any{}

	if v := update.UpdatedTs; v != nil {
		set, args = append(set, "updated_ts = ?"), append(args, *v)
	}
	if v := update.Filename; v != nil {
		set, args = append(set, "filename = ?"), append(args, *v)
	}
	if v := update.InternalPath; v != nil {
		set, args = append(set, "internal_path = ?"), append(args, *v)
	}
	if v := update.MemoID; v != nil {
		set, args = append(set, "memo_id = ?"), append(args, *v)
	}
	if update.UnbindMemo {
		set = append(set, "memo_id = NULL")
	}
	if v := update.Blob; v != nil {
		set, args = append(set, "blob = ?"), append(args, v)
	}

	args = append(args, update.ID)
	fields := []string{"id", "filename", "external_link", "type", "size", "creator_id", "created_ts", "updated_ts", "internal_path"}
	stmt := `
		UPDATE resource
		SET ` + strings.Join(set, ", ") + `
		WHERE id = ?
		RETURNING ` + strings.Join(fields, ", ")
	resource := store.Resource{}
	dests := []any{
		&resource.ID,
		&resource.Filename,
		&resource.ExternalLink,
		&resource.Type,
		&resource.Size,
		&resource.CreatorID,
		&resource.CreatedTs,
		&resource.UpdatedTs,
		&resource.InternalPath,
	}
	if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(dests...); err != nil {
		return nil, err
	}

	return &resource, nil
}

func (d *Driver) DeleteResource(ctx context.Context, delete *store.DeleteResource) error {
	stmt := `
		DELETE FROM resource
		WHERE id = ?
	`
	result, err := d.db.ExecContext(ctx, stmt, delete.ID)
	if err != nil {
		return err
	}
	if _, err := result.RowsAffected(); err != nil {
		return err
	}

	return nil
}
