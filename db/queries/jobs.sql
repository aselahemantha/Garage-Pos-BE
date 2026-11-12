-- name: CreateJobCard :one
INSERT INTO job_cards (
    job_no,
    customer_id,
    vehicle_id,
    advisor_id,
    mileage_in,
    complaint,
    notes,
    status,
    promised_by
)
VALUES (
    'JOB-' || LPAD(NEXTVAL('job_no_seq')::text, 6, '0'),
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, job_no, customer_id, vehicle_id, advisor_id, mileage_in, complaint, notes, status, promised_by, started_at, completed_at, delivered_at, created_at, updated_at;

-- name: GetJobCardByID :one
SELECT 
    j.id, 
    j.job_no, 
    j.customer_id, 
    c.name AS customer_name,
    c.phone AS customer_phone,
    j.vehicle_id, 
    v.plate_no,
    v.make AS vehicle_make,
    v.model AS vehicle_model,
    v.year AS vehicle_year,
    j.advisor_id, 
    u.name AS advisor_name,
    j.mileage_in, 
    j.complaint, 
    j.notes, 
    j.status, 
    j.promised_by, 
    j.started_at, 
    j.completed_at, 
    j.delivered_at, 
    j.created_at, 
    j.updated_at
FROM job_cards j
JOIN customers c ON j.customer_id = c.id
JOIN vehicles v ON j.vehicle_id = v.id
JOIN users u ON j.advisor_id = u.id
WHERE j.id = $1;

-- name: ListJobCards :many
SELECT 
    j.id, 
    j.job_no, 
    j.customer_id, 
    c.name AS customer_name,
    c.phone AS customer_phone,
    j.vehicle_id, 
    v.plate_no,
    v.make AS vehicle_make,
    v.model AS vehicle_model,
    j.advisor_id, 
    u.name AS advisor_name,
    j.mileage_in, 
    j.complaint, 
    j.status, 
    j.promised_by, 
    j.started_at, 
    j.completed_at, 
    j.delivered_at, 
    j.created_at
FROM job_cards j
JOIN customers c ON j.customer_id = c.id
JOIN vehicles v ON j.vehicle_id = v.id
JOIN users u ON j.advisor_id = u.id
WHERE (sqlc.narg('status')::text IS NULL OR j.status = sqlc.narg('status'))
ORDER BY j.created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBoardJobs :many
SELECT 
    j.id, 
    j.job_no, 
    j.customer_id, 
    c.name AS customer_name,
    c.phone AS customer_phone,
    j.vehicle_id, 
    v.plate_no,
    v.make AS vehicle_make,
    v.model AS vehicle_model,
    v.year AS vehicle_year,
    j.advisor_id, 
    j.mileage_in, 
    j.complaint, 
    j.notes,
    j.status, 
    j.promised_by, 
    j.started_at, 
    j.completed_at, 
    j.delivered_at, 
    j.created_at
FROM job_cards j
JOIN customers c ON j.customer_id = c.id
JOIN vehicles v ON j.vehicle_id = v.id
ORDER BY j.created_at ASC;

-- name: UpdateJobCard :one
UPDATE job_cards
SET mileage_in = COALESCE(sqlc.narg('mileage_in'), mileage_in),
    complaint = COALESCE(sqlc.narg('complaint'), complaint),
    notes = COALESCE(sqlc.narg('notes'), notes),
    promised_by = COALESCE(sqlc.narg('promised_by'), promised_by),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, job_no, customer_id, vehicle_id, advisor_id, mileage_in, complaint, notes, status, promised_by, started_at, completed_at, delivered_at, created_at, updated_at;

-- name: UpdateJobStatus :one
UPDATE job_cards
SET status = $2,
    started_at = CASE WHEN $2 = 'in_progress' AND started_at IS NULL THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $2 = 'completed' AND completed_at IS NULL THEN NOW() ELSE completed_at END,
    delivered_at = CASE WHEN $2 = 'delivered' AND delivered_at IS NULL THEN NOW() ELSE delivered_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING id, job_no, customer_id, vehicle_id, advisor_id, mileage_in, complaint, notes, status, promised_by, started_at, completed_at, delivered_at, created_at, updated_at;

-- name: CreateJobItem :one
INSERT INTO job_items (
    job_card_id,
    item_type,
    service_id,
    part_id,
    description,
    qty,
    unit_price,
    discount,
    line_total,
    mechanic_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, job_card_id, item_type, service_id, part_id, description, qty, unit_price, discount, line_total, mechanic_id, created_at, updated_at;

-- name: ListJobItems :many
SELECT 
    ji.id, 
    ji.job_card_id, 
    ji.item_type, 
    ji.service_id, 
    ji.part_id, 
    ji.description, 
    ji.qty, 
    ji.unit_price, 
    ji.discount, 
    ji.line_total, 
    ji.mechanic_id, 
    u.name AS mechanic_name,
    ji.created_at, 
    ji.updated_at
FROM job_items ji
LEFT JOIN users u ON ji.mechanic_id = u.id
WHERE ji.job_card_id = $1
ORDER BY ji.id ASC;

-- name: GetJobItemByID :one
SELECT id, job_card_id, item_type, service_id, part_id, description, qty, unit_price, discount, line_total, mechanic_id, created_at, updated_at
FROM job_items
WHERE id = $1;

-- name: DeleteJobItem :exec
DELETE FROM job_items
WHERE id = $1;
