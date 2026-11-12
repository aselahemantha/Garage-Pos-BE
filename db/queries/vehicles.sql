-- name: GetVehicleByID :one
SELECT id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at
FROM vehicles
WHERE id = $1;

-- name: GetVehicleByPlate :one
SELECT id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at
FROM vehicles
WHERE plate_no = $1;

-- name: ListVehiclesByCustomerID :many
SELECT id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at
FROM vehicles
WHERE customer_id = $1
ORDER BY id DESC;

-- name: SearchVehicles :many
SELECT id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at
FROM vehicles
WHERE ($1::text = '' OR plate_no ILIKE '%' || $1 || '%')
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: CountVehicles :one
SELECT COUNT(*)
FROM vehicles
WHERE ($1::text = '' OR plate_no ILIKE '%' || $1 || '%');

-- name: CreateVehicle :one
INSERT INTO vehicles (customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at;

-- name: UpdateVehicle :one
UPDATE vehicles
SET make = COALESCE(sqlc.narg('make'), make),
    model = COALESCE(sqlc.narg('model'), model),
    year = COALESCE(sqlc.narg('year'), year),
    vin = COALESCE(sqlc.narg('vin'), vin),
    fuel_type = COALESCE(sqlc.narg('fuel_type'), fuel_type),
    color = COALESCE(sqlc.narg('color'), color),
    last_mileage = COALESCE(sqlc.narg('last_mileage'), last_mileage),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, customer_id, plate_no, make, model, year, vin, fuel_type, color, last_mileage, created_at, updated_at;
