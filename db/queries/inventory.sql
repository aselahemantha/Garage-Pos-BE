-- name: ListParts :many
SELECT 
    p.id, 
    p.sku, 
    p.name, 
    p.category, 
    p.unit, 
    p.cost_price, 
    p.sell_price, 
    p.reorder_level, 
    p.is_active, 
    p.created_at, 
    p.updated_at,
    COALESCE(SUM(sm.qty), 0)::numeric(12,3) AS stock_on_hand
FROM parts p
LEFT JOIN stock_movements sm ON p.id = sm.part_id
WHERE ($1::text = '' OR p.name ILIKE '%' || $1 || '%' OR p.sku ILIKE '%' || $1 || '%')
  AND (sqlc.narg('category')::text IS NULL OR p.category = sqlc.narg('category'))
  AND (sqlc.narg('is_active')::boolean IS NULL OR p.is_active = sqlc.narg('is_active'))
GROUP BY p.id
ORDER BY p.name ASC;

-- name: GetPartByID :one
SELECT 
    p.id, 
    p.sku, 
    p.name, 
    p.category, 
    p.unit, 
    p.cost_price, 
    p.sell_price, 
    p.reorder_level, 
    p.is_active, 
    p.created_at, 
    p.updated_at,
    COALESCE(SUM(sm.qty), 0)::numeric(12,3) AS stock_on_hand
FROM parts p
LEFT JOIN stock_movements sm ON p.id = sm.part_id
WHERE p.id = $1
GROUP BY p.id;

-- name: GetPartBySKU :one
SELECT id, sku, name, category, unit, cost_price, sell_price, reorder_level, is_active, created_at, updated_at
FROM parts
WHERE sku = $1;

-- name: CreatePart :one
INSERT INTO parts (sku, name, category, unit, cost_price, sell_price, reorder_level, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, sku, name, category, unit, cost_price, sell_price, reorder_level, is_active, created_at, updated_at;

-- name: UpdatePart :one
UPDATE parts
SET name = COALESCE(sqlc.narg('name'), name),
    category = COALESCE(sqlc.narg('category'), category),
    unit = COALESCE(sqlc.narg('unit'), unit),
    cost_price = COALESCE(sqlc.narg('cost_price'), cost_price),
    sell_price = COALESCE(sqlc.narg('sell_price'), sell_price),
    reorder_level = COALESCE(sqlc.narg('reorder_level'), reorder_level),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, sku, name, category, unit, cost_price, sell_price, reorder_level, is_active, created_at, updated_at;

-- name: AddStockMovement :one
INSERT INTO stock_movements (part_id, qty, type, ref_type, ref_id, unit_cost, note, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, part_id, qty, type, ref_type, ref_id, unit_cost, note, created_by, created_at;

-- name: GetStockOnHand :one
SELECT COALESCE(SUM(qty), 0)::numeric(12,3)
FROM stock_movements
WHERE part_id = $1;
