-- name: NextInvoiceNo :one
SELECT ('INV-' || LPAD(nextval('invoice_no_seq')::TEXT, 6, '0'))::TEXT AS invoice_no;

-- name: NextCashSessionNo :one
SELECT ('CS-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(nextval('cash_session_no_seq')::TEXT, 4, '0'))::TEXT AS session_no;

-- name: CreateInvoice :one
INSERT INTO invoices (
    invoice_no,
    job_card_id,
    customer_id,
    vehicle_id,
    cash_session_id,
    status,
    subtotal,
    discount_total,
    tax_bps,
    tax_total,
    total,
    paid_total,
    balance,
    issued_at,
    due_at,
    notes,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetInvoiceByID :one
SELECT 
    i.*,
    c.name AS customer_name,
    c.phone AS customer_phone,
    c.address AS customer_address,
    v.plate_no,
    v.make AS vehicle_make,
    v.model AS vehicle_model,
    v.color AS vehicle_color,
    jc.job_no,
    u.name AS created_by_name
FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN vehicles v ON v.id = i.vehicle_id
LEFT JOIN job_cards jc ON jc.id = i.job_card_id
JOIN users u ON u.id = i.created_by
WHERE i.id = $1;

-- name: GetInvoiceByJobCardID :one
SELECT 
    i.*,
    c.name AS customer_name,
    c.phone AS customer_phone,
    v.plate_no,
    jc.job_no
FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN vehicles v ON v.id = i.vehicle_id
LEFT JOIN job_cards jc ON jc.id = i.job_card_id
WHERE i.job_card_id = $1 AND i.status != 'void'
LIMIT 1;

-- name: ListInvoices :many
SELECT 
    i.*,
    c.name AS customer_name,
    c.phone AS customer_phone,
    v.plate_no,
    jc.job_no
FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN vehicles v ON v.id = i.vehicle_id
LEFT JOIN job_cards jc ON jc.id = i.job_card_id
WHERE 
    ($1::TEXT IS NULL OR i.status = $1)
    AND (
        $2::TEXT IS NULL 
        OR i.invoice_no ILIKE '%' || $2 || '%'
        OR c.name ILIKE '%' || $2 || '%'
        OR c.phone ILIKE '%' || $2 || '%'
        OR v.plate_no ILIKE '%' || $2 || '%'
    )
ORDER BY i.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountInvoices :one
SELECT COUNT(*) FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN vehicles v ON v.id = i.vehicle_id
WHERE 
    ($1::TEXT IS NULL OR i.status = $1)
    AND (
        $2::TEXT IS NULL 
        OR i.invoice_no ILIKE '%' || $2 || '%'
        OR c.name ILIKE '%' || $2 || '%'
        OR c.phone ILIKE '%' || $2 || '%'
        OR v.plate_no ILIKE '%' || $2 || '%'
    );

-- name: UpdateInvoicePaymentProgress :one
UPDATE invoices
SET 
    paid_total = $2,
    balance = $3,
    status = $4,
    paid_at = CASE WHEN $4 = 'paid' THEN NOW() ELSE paid_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: VoidInvoice :one
UPDATE invoices
SET 
    status = 'void',
    voided_at = NOW(),
    void_reason = $2,
    voided_by = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateInvoiceItem :one
INSERT INTO invoice_items (
    invoice_id,
    item_type,
    service_id,
    part_id,
    description,
    qty,
    unit_price,
    discount,
    line_total,
    mechanic_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: ListInvoiceItemsByInvoiceID :many
SELECT * FROM invoice_items
WHERE invoice_id = $1
ORDER BY id ASC;

-- name: CreatePayment :one
INSERT INTO payments (
    invoice_id,
    cash_session_id,
    payment_method,
    amount,
    tendered,
    change_given,
    reference_no,
    notes,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: ListPaymentsByInvoiceID :many
SELECT 
    p.*,
    u.name AS created_by_name
FROM payments p
JOIN users u ON u.id = p.created_by
WHERE p.invoice_id = $1
ORDER BY p.created_at ASC;

-- name: GetActiveCashSession :one
SELECT * FROM cash_sessions
WHERE status = 'open'
ORDER BY opened_at DESC
LIMIT 1;

-- name: CreateCashSession :one
INSERT INTO cash_sessions (
    session_no,
    user_id,
    opening_float,
    expected_cash,
    status,
    notes
) VALUES (
    $1, $2, $3, $3, 'open', $4
) RETURNING *;

-- name: IncrementCashSessionExpectedCash :one
UPDATE cash_sessions
SET 
    expected_cash = expected_cash + $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CloseCashSession :one
UPDATE cash_sessions
SET 
    closed_at = NOW(),
    closing_counted = $2,
    difference = $2 - expected_cash,
    status = 'closed',
    notes = COALESCE($3, notes),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetCashSessionByID :one
SELECT 
    cs.*,
    u.name AS user_name
FROM cash_sessions cs
JOIN users u ON u.id = cs.user_id
WHERE cs.id = $1;

-- name: ListCashSessions :many
SELECT 
    cs.*,
    u.name AS user_name
FROM cash_sessions cs
JOIN users u ON u.id = cs.user_id
ORDER BY cs.opened_at DESC
LIMIT $1 OFFSET $2;
