CREATE TABLE IF NOT EXISTS users (
                                     uid varchar(36) NOT NULL PRIMARY KEY,
    cart_id UUID NOT NULL,
    email TEXT NOT NULL,
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
    count integer NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT false
    );


CREATE TABLE IF NOT EXISTS cart (
                                    cart_id UUID NOT NULL PRIMARY KEY,
                                    user_id VARCHAR(36) NOT NULL REFERENCES users(uid) ON DELETE CASCADE
    );


CREATE TABLE IF NOT EXISTS cart_items (
                                          item_id SERIAL PRIMARY KEY,
                                          cart_id UUID NOT NULL REFERENCES cart(cart_id) ON DELETE CASCADE,
    book_id varchar(36) NOT NULL REFERENCES books(bid) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0)
    );

