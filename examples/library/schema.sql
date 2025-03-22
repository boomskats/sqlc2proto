-- Library Management System Schema

-- Books Table
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    isbn TEXT UNIQUE NOT NULL,
    published_on DATE NOT NULL,
    page_count INTEGER NOT NULL,
    genre TEXT NOT NULL,
    summary TEXT,
    in_stock BOOLEAN NOT NULL DEFAULT TRUE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Members Table
CREATE TABLE members (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT,
    join_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expiry_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Loans Table
CREATE TABLE loans (
    id SERIAL PRIMARY KEY,
    book_id INTEGER NOT NULL REFERENCES books(id),
    member_id INTEGER NOT NULL REFERENCES members(id),
    loan_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_date TIMESTAMPTZ NOT NULL,
    returned_date TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('active', 'returned', 'overdue', 'lost')),
    
    CONSTRAINT valid_loan_dates CHECK (
        loan_date <= due_date AND
        (returned_date IS NULL OR returned_date >= loan_date)
    )
);

-- Indexes for performance
CREATE INDEX idx_books_genre ON books(genre);
CREATE INDEX idx_books_author ON books(author);
CREATE INDEX idx_loans_book_id ON loans(book_id);
CREATE INDEX idx_loans_member_id ON loans(member_id);
CREATE INDEX idx_loans_status ON loans(status);
