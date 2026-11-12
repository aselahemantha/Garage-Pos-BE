-- name: ListServiceCategories :many
SELECT id, name, sort_order, created_at, updated_at
FROM service_categories
ORDER BY sort_order ASC, name ASC;

-- name: CreateServiceCategory :one
INSERT INTO service_categories (name, sort_order)
VALUES ($1, $2)
RETURNING id, name, sort_order, created_at, updated_at;

-- name: ListServices :many
SELECT s.id, s.category_id, sc.name AS category_name, s.name, s.description, s.price, s.est_minutes, s.is_active, s.created_at, s.updated_at
FROM services s
JOIN service_categories sc ON s.category_id = sc.id
WHERE (sqlc.narg('category_id')::bigint IS NULL OR s.category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('is_active')::boolean IS NULL OR s.is_active = sqlc.narg('is_active'))
ORDER BY sc.sort_order ASC, s.name ASC;

-- name: GetServiceByID :one
SELECT s.id, s.category_id, sc.name AS category_name, s.name, s.description, s.price, s.est_minutes, s.is_active, s.created_at, s.updated_at
FROM services s
JOIN service_categories sc ON s.category_id = sc.id
WHERE s.id = $1;

-- name: CreateService :one
INSERT INTO services (category_id, name, description, price, est_minutes, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, category_id, name, description, price, est_minutes, is_active, created_at, updated_at;

-- name: UpdateService :one
UPDATE services
SET category_id = COALESCE(sqlc.narg('category_id'), category_id),
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    price = COALESCE(sqlc.narg('price'), price),
    est_minutes = COALESCE(sqlc.narg('est_minutes'), est_minutes),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, category_id, name, description, price, est_minutes, is_active, created_at, updated_at;
