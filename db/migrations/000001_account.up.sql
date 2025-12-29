CREATE SCHEMA account;

CREATE TABLE
    account.user (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        fname VARCHAR(255) NOT NULL,
        lname VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        user_password BYTEA NOT NULL,
        account_status VARCHAR(50) NOT NULL,
        email_verified BOOLEAN NOT NULL DEFAULT false,
        role VARCHAR(50) NOT NULL,
        profile_picture_url TEXT DEFAULT NULL,
        verified_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE
    account.token (
        token TEXT PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL,
        email TEXT NOT NULL,
        purpose TEXT NOT NULL,
        used BOOLEAN NOT NULL DEFAULT FALSE,
        revoked BOOLEAN NOT NULL DEFAULT FALSE,
        expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

CREATE TABLE
    account.user_session (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        token TEXT NOT NULL,
        user_id UUID NOT NULL,
        role VARCHAR(50) NOT NULL,
        user_agent TEXT NOT NULL,
        ip_address TEXT NOT NULL,
        revoked BOOLEAN NOT NULL DEFAULT FALSE,
        revoked_at TIMESTAMPTZ DEFAULT NULL,
        expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );