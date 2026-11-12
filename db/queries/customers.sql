-- name: GetCustomerByID :one
SELECT id, name, phone, email, address, notes, is_active, created_at, updated_at
FROM customers
WHERE id = $1;

-- name: GetCustomerByPhone :one
SELECT id, name, phone, email, address, notes, is_active, created_at, updated_at
FROM customers
WHERE phone = $1;

-- name: SearchCustomers :many
SELECT id, name, phone, email, address, notes, is_active, created_at, updated_at
FROM customers
WHERE ($1::text = '' OR phone ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
  AND is_active = true
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: CountCustomers :one
SELECT COUNT(*)
FROM customers
WHERE ($1::text = '' OR phone ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
  AND is_active = true;

-- name: CreateCustomer :one
INSERT INTO customers (name, phone, email, address, notes, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, phone, email, address, notes, is_active, created_at, updated_at;

-- name: UpdateCustomer :one
UPDATE customers
SET name = COALESCE(sqlc.narg('name'), name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    email = COALESCE(sqlc.narg('email'), email),
    address = COALESCE(sqlc.narg('address'), address),
    notes = COALESCE(sqlc.narg('notes'), notes),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, name, phone, email, address, notes, is_active, created_at, updated_at;
