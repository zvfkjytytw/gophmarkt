CREATE SCHEMA IF NOT EXISTS gophmarkt;

-- Type of the order statuses
CREATE TYPE gophmarkt.order_status AS ENUM (
    'NEW',
    'PROCESSING',
    'INVALID',
    'PROCEESED'
);

-- USERS

-- Table of users
CREATE TABLE IF NOT EXISTS gophmarkt.users (
    login    text not null, -- username
    password text not null  -- password
);

-- Set the user as the defining one
ALTER TABLE gophmarkt.users ADD PRIMARY KEY (login);

-- Index to optimize the search for user
CREATE UNIQUE INDEX idx_gophmarkt_users ON gophmarkt.users (login);

-- BALANCE
-- Table of balance
CREATE TABLE IF NOT EXISTS gophmarkt.balance (
    login     text not null, -- username
    current   real not null, -- accumulated points
    withdrawn real           -- drawn points
);

-- Set the user as the defining one
ALTER TABLE gophmarkt.balance ADD PRIMARY KEY (login);

-- Index to optimize the search for balance
CREATE UNIQUE INDEX idx_gophmarkt_balance ON gophmarkt.balance (login);

-- ORDERS
-- Table of orders
CREATE TABLE IF NOT EXISTS gophmarkt.orders (
    order_id    text not null,                   -- order id 
    login       text not null,                   -- username
    status      gophmarkt.order_status not null, -- order status
    accrual     real,                            -- order points
    date_upload timestamp not null,              -- order upload date
    date_update timestamp not null               -- order last update date
);

-- Set the user as the defining one
ALTER TABLE gophmarkt.orders ADD PRIMARY KEY (order_id);

-- WITHDRAWALS
-- Table of withdrawals
CREATE TABLE IF NOT EXISTS gophmarkt.withdrawals (
    order_id text not null,     -- order id 
    login    text not null,     -- username
    count    real not null,     -- deducted points
    offdate  timestamp not null -- date of debiting
);

-- Set the user as the defining one
ALTER TABLE gophmarkt.withdrawals ADD PRIMARY KEY (order_id);
