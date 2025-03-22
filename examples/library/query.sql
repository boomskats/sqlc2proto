-- name: GetBook :one
SELECT * FROM books
WHERE id = $1;

-- name: ListBooks :many
SELECT * FROM books
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: SearchBooks :many
SELECT * FROM books
WHERE 
    ($1::text IS NULL OR title ILIKE '%' || $1 || '%') AND
    ($2::text IS NULL OR author ILIKE '%' || $2 || '%') AND
    ($3::text IS NULL OR genre = $3) AND
    ($4::boolean IS NULL OR in_stock = $4)
ORDER BY title
LIMIT $5 OFFSET $6;

-- name: CreateBook :one
INSERT INTO books (
    title, author, isbn, published_on, page_count, genre, summary, in_stock
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetMember :one
SELECT * FROM members
WHERE id = $1;

-- name: ListMembers :many
SELECT * FROM members
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: CreateMember :one
INSERT INTO members (
    name, email, phone, expiry_date
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetLoan :one
SELECT * FROM loans
WHERE id = $1;

-- name: ListActiveLoansByMember :many
SELECT * FROM loans
WHERE member_id = $1 AND status = 'active'
ORDER BY due_date;

-- name: CreateLoan :one
INSERT INTO loans (
    book_id, member_id, due_date, status
) VALUES (
    $1, $2, $3, 'active'
)
RETURNING *;

-- name: ReturnBook :one
UPDATE loans
SET returned_date = NOW(), status = 'returned'
WHERE id = $1 AND status = 'active'
RETURNING *;
