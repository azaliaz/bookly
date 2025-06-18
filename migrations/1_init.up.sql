CREATE TABLE IF NOT EXISTS users (
                                     uid varchar(36) NOT NULL PRIMARY KEY,
    cart_id UUID NOT NULL,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    lastname TEXT NOT NULL,
    pass TEXT NOT NULL,
    age integer,
    role TEXT NOT NULL DEFAULT 'user'
    );

CREATE UNIQUE INDEX IF NOT EXISTS email_id ON users (email);


CREATE TABLE IF NOT EXISTS books (
                                     bid varchar(36) NOT NULL PRIMARY KEY,
    lable TEXT NOT NULL,
    author TEXT NOT NULL,
    "desc" TEXT NOT NULL,
    age integer NOT NULL,
    genre TEXT DEFAULT 'Без жанра',
    rating INTEGER DEFAULT 0,
    cover_url TEXT,
    pdf_url TEXT
    );


CREATE TABLE IF NOT EXISTS cart (
                                    cart_id UUID NOT NULL PRIMARY KEY,
                                    user_id VARCHAR(36) NOT NULL REFERENCES users(uid) ON DELETE CASCADE
    );


CREATE TABLE IF NOT EXISTS cart_items (
                                          item_id UUID NOT NULL PRIMARY KEY,
                                          cart_id UUID NOT NULL REFERENCES cart(cart_id) ON DELETE CASCADE,
    book_id varchar(36) NOT NULL REFERENCES books(bid) ON DELETE CASCADE
    );


CREATE TABLE IF NOT EXISTS feedbacks (
                                         feedback_id UUID NOT NULL PRIMARY KEY,
                                         user_id VARCHAR(36) NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
    book_id VARCHAR(36) NOT NULL REFERENCES books(bid) ON DELETE CASCADE,
    text TEXT NOT NULL,
    create_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
