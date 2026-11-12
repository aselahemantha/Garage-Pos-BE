-- name: GetRevenueSummary :one
SELECT 
    COALESCE(SUM(p.amount), 0)::BIGINT AS total_collected,
    COALESCE(SUM(CASE WHEN p.payment_method = 'cash' THEN p.amount ELSE 0 END), 0)::BIGINT AS cash_collected,
    COALESCE(SUM(CASE WHEN p.payment_method = 'card' THEN p.amount ELSE 0 END), 0)::BIGINT AS card_collected,
    COALESCE(SUM(CASE WHEN p.payment_method = 'bank_transfer' THEN p.amount ELSE 0 END), 0)::BIGINT AS bank_collected,
    COUNT(DISTINCT p.invoice_id)::BIGINT AS paid_invoices_count
FROM payments p
WHERE p.created_at >= $1 AND p.created_at <= $2;

-- name: GetInvoicedSummary :one
SELECT 
    COALESCE(SUM(i.total), 0)::BIGINT AS total_invoiced,
    COALESCE(SUM(i.subtotal), 0)::BIGINT AS total_subtotal,
    COALESCE(SUM(i.discount_total), 0)::BIGINT AS total_discount,
    COALESCE(SUM(i.tax_total), 0)::BIGINT AS total_tax,
    COUNT(i.id)::BIGINT AS invoice_count
FROM invoices i
WHERE i.status != 'void'
  AND i.issued_at >= $1 AND i.issued_at <= $2;

-- name: GetOutstandingSummary :one
SELECT 
    COALESCE(SUM(i.balance), 0)::BIGINT AS total_outstanding,
    COUNT(i.id)::BIGINT AS outstanding_count
FROM invoices i
WHERE i.status IN ('issued', 'partially_paid')
  AND i.balance > 0;

-- name: GetCompletedJobsSummary :one
SELECT 
    COUNT(j.id)::BIGINT AS completed_jobs_count
FROM job_cards j
WHERE j.status IN ('completed', 'delivered')
  AND j.completed_at >= $1 AND j.completed_at <= $2;

-- name: GetDailyRevenueTrend :many
SELECT 
    TO_CHAR(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
    COALESCE(SUM(p.amount), 0)::BIGINT AS total_amount,
    COALESCE(SUM(CASE WHEN p.payment_method = 'cash' THEN p.amount ELSE 0 END), 0)::BIGINT AS cash_amount,
    COALESCE(SUM(CASE WHEN p.payment_method = 'card' THEN p.amount ELSE 0 END), 0)::BIGINT AS card_amount,
    COALESCE(SUM(CASE WHEN p.payment_method = 'bank_transfer' THEN p.amount ELSE 0 END), 0)::BIGINT AS bank_amount
FROM payments p
WHERE p.created_at >= $1 AND p.created_at <= $2
GROUP BY day
ORDER BY day ASC;

-- name: GetPaymentMethodsBreakdown :many
SELECT 
    p.payment_method,
    COALESCE(SUM(p.amount), 0)::BIGINT AS total_amount,
    COUNT(p.id)::BIGINT AS payment_count
FROM payments p
WHERE p.created_at >= $1 AND p.created_at <= $2
GROUP BY p.payment_method
ORDER BY total_amount DESC;

-- name: GetRevenueCategorySplit :one
SELECT 
    COALESCE(SUM(CASE WHEN ii.item_type = 'labor' THEN ii.line_total ELSE 0 END), 0)::BIGINT AS labor_total,
    COALESCE(SUM(CASE WHEN ii.item_type = 'part' THEN ii.line_total ELSE 0 END), 0)::BIGINT AS parts_total,
    COALESCE(SUM(CASE WHEN ii.item_type = 'custom' THEN ii.line_total ELSE 0 END), 0)::BIGINT AS custom_total
FROM invoice_items ii
JOIN invoices i ON i.id = ii.invoice_id
WHERE i.status != 'void'
  AND i.issued_at >= $1 AND i.issued_at <= $2;

-- name: GetTopServices :many
SELECT 
    s.id,
    s.name,
    COUNT(ii.id)::BIGINT AS times_performed,
    COALESCE(SUM(ii.line_total), 0)::BIGINT AS total_revenue
FROM invoice_items ii
JOIN services s ON s.id = ii.service_id
JOIN invoices i ON i.id = ii.invoice_id
WHERE ii.item_type = 'labor'
  AND i.status != 'void'
  AND i.issued_at >= $1 AND i.issued_at <= $2
GROUP BY s.id, s.name
ORDER BY total_revenue DESC
LIMIT $3;

-- name: GetTopParts :many
SELECT 
    p.id,
    p.name,
    p.sku,
    COALESCE(SUM(ii.qty), 0)::NUMERIC AS qty_sold,
    COALESCE(SUM(ii.line_total), 0)::BIGINT AS total_revenue
FROM invoice_items ii
JOIN parts p ON p.id = ii.part_id
JOIN invoices i ON i.id = ii.invoice_id
WHERE ii.item_type = 'part'
  AND i.status != 'void'
  AND i.issued_at >= $1 AND i.issued_at <= $2
GROUP BY p.id, p.name, p.sku
ORDER BY total_revenue DESC
LIMIT $3;

-- name: GetTopOutstandingInvoices :many
SELECT 
    i.id,
    i.invoice_no,
    i.customer_id,
    c.name AS customer_name,
    c.phone AS customer_phone,
    v.plate_no,
    i.total,
    i.paid_total,
    i.balance,
    i.issued_at
FROM invoices i
JOIN customers c ON c.id = i.customer_id
LEFT JOIN vehicles v ON v.id = i.vehicle_id
WHERE i.status IN ('issued', 'partially_paid')
  AND i.balance > 0
ORDER BY i.balance DESC
LIMIT $1;

-- name: GetDayCashSessions :many
SELECT 
    cs.id,
    cs.session_no,
    cs.user_id,
    u.name AS user_name,
    cs.opened_at,
    cs.closed_at,
    cs.opening_float,
    cs.closing_counted,
    cs.expected_cash,
    cs.difference,
    cs.status
FROM cash_sessions cs
JOIN users u ON u.id = cs.user_id
WHERE cs.opened_at >= $1 AND cs.opened_at <= $2
ORDER BY cs.opened_at ASC;
